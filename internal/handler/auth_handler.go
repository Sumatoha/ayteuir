package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/ayteuir/backend/internal/config"
	"github.com/ayteuir/backend/internal/middleware"
	"github.com/ayteuir/backend/internal/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuthHandler struct {
	authService *service.AuthService
	userService *service.UserService
	cfg         *config.Config
}

func NewAuthHandler(authService *service.AuthService, userService *service.UserService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userService: userService,
		cfg:         cfg,
	}
}

func (h *AuthHandler) InitiateOAuth(w http.ResponseWriter, r *http.Request) {
	state := generateState()

	authURL := h.authService.GetAuthorizationURL(state)

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	frontendURL := h.cfg.App.FrontendURL

	if code == "" {
		errorParam := r.URL.Query().Get("error")
		errorDesc := r.URL.Query().Get("error_description")
		if errorParam != "" {
			redirectURL := fmt.Sprintf("%s/callback?error=%s&error_description=%s", frontendURL, errorParam, errorDesc)
			http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
			return
		}
		redirectURL := fmt.Sprintf("%s/callback?error=missing_code&error_description=Authorization+code+is+required", frontendURL)
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		return
	}

	_, token, err := h.authService.HandleCallback(r.Context(), code)
	if err != nil {
		redirectURL := fmt.Sprintf("%s/callback?error=auth_failed&error_description=%s", frontendURL, err.Error())
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		return
	}

	redirectURL := fmt.Sprintf("%s/callback?token=%s", frontendURL, token)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	userIDStr := middleware.GetUserID(r.Context())
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	if err := h.authService.RefreshThreadsToken(r.Context(), userID); err != nil {
		Error(w, http.StatusInternalServerError, "REFRESH_FAILED", err.Error())
		return
	}

	newToken, err := h.authService.GenerateToken(userIDStr)
	if err != nil {
		Error(w, http.StatusInternalServerError, "TOKEN_GEN_FAILED", err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]string{
		"token": newToken,
	})
}

func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := middleware.GetUserID(r.Context())
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	user, err := h.userService.GetByID(r.Context(), userID)
	if err != nil {
		Error(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		return
	}

	JSON(w, http.StatusOK, user)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]string{
		"message": "Logged out successfully",
	})
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
