package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ayteuir/backend/internal/config"
	"github.com/ayteuir/backend/internal/domain"
	"github.com/ayteuir/backend/internal/pkg/logger"
	"github.com/ayteuir/backend/internal/pkg/threads"
	"github.com/ayteuir/backend/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuthService struct {
	userRepo      repository.UserRepository
	threadsClient *threads.Client
	cfg           *config.Config
}

type JWTClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func NewAuthService(userRepo repository.UserRepository, threadsClient *threads.Client, cfg *config.Config) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		threadsClient: threadsClient,
		cfg:           cfg,
	}
}

func (s *AuthService) GetAuthorizationURL(state string) string {
	return s.threadsClient.GetAuthorizationURL(state)
}

func (s *AuthService) HandleCallback(ctx context.Context, code string) (*domain.User, string, error) {
	tokenResp, err := s.threadsClient.ExchangeCodeForToken(ctx, code)
	if err != nil {
		return nil, "", fmt.Errorf("failed to exchange code: %w", err)
	}

	longLivedResp, err := s.threadsClient.ExchangeForLongLivedToken(ctx, tokenResp.AccessToken)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get long-lived token, using short-lived")
		longLivedResp = &threads.LongLivedTokenResponse{
			AccessToken: tokenResp.AccessToken,
			ExpiresIn:   tokenResp.ExpiresIn,
		}
	}

	profile, err := s.threadsClient.GetUserProfile(ctx, longLivedResp.AccessToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user profile: %w", err)
	}

	user, err := s.userRepo.GetByThreadsUserID(ctx, profile.ID)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return nil, "", fmt.Errorf("failed to check existing user: %w", err)
		}

		user = domain.NewUser(profile.ID, profile.Username, profile.Name, profile.ThreadsProfileURL)
	}

	encryptedToken, err := s.encryptToken(longLivedResp.AccessToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to encrypt token: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(longLivedResp.ExpiresIn) * time.Second)
	user.SetTokens(encryptedToken, "", expiresAt)
	user.Username = profile.Username
	user.DisplayName = profile.Name
	user.ProfilePictureURL = profile.ThreadsProfileURL

	if user.ID.IsZero() {
		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, "", fmt.Errorf("failed to create user: %w", err)
		}
	} else {
		if err := s.userRepo.Update(ctx, user); err != nil {
			return nil, "", fmt.Errorf("failed to update user: %w", err)
		}
	}

	jwtToken, err := s.GenerateToken(user.ID.Hex())
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	return user, jwtToken, nil
}

func (s *AuthService) GenerateToken(userID string) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.JWTExpiry())),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "ayteuir",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Security.JWTSecret))
}

func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.Security.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	return claims, nil
}

func (s *AuthService) RefreshThreadsToken(ctx context.Context, userID primitive.ObjectID) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	decryptedToken, err := s.decryptToken(user.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to decrypt token: %w", err)
	}

	refreshResp, err := s.threadsClient.RefreshToken(ctx, decryptedToken)
	if err != nil {
		return fmt.Errorf("failed to refresh Threads token: %w", err)
	}

	encryptedToken, err := s.encryptToken(refreshResp.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(refreshResp.ExpiresIn) * time.Second)
	user.SetTokens(encryptedToken, "", expiresAt)

	return s.userRepo.Update(ctx, user)
}

func (s *AuthService) GetDecryptedAccessToken(ctx context.Context, userID primitive.ObjectID) (string, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}

	if user.IsTokenExpired() {
		if err := s.RefreshThreadsToken(ctx, userID); err != nil {
			return "", fmt.Errorf("token expired and refresh failed: %w", err)
		}
		user, _ = s.userRepo.GetByID(ctx, userID)
	}

	return s.decryptToken(user.AccessToken)
}

func (s *AuthService) encryptToken(plaintext string) (string, error) {
	block, err := aes.NewCipher([]byte(s.cfg.Security.EncryptionKey))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *AuthService) decryptToken(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(s.cfg.Security.EncryptionKey))
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(data) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
