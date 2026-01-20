package service

import (
	"context"

	"github.com/ayteuir/backend/internal/domain"
	"github.com/ayteuir/backend/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

func (s *UserService) GetByID(ctx context.Context, userID primitive.ObjectID) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

func (s *UserService) UpdateSettings(ctx context.Context, userID primitive.ObjectID, settings domain.UserSettings) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	user.Settings = settings
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) ToggleAutoReply(ctx context.Context, userID primitive.ObjectID, enabled bool) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	user.AutoReplyEnabled = enabled
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) Delete(ctx context.Context, userID primitive.ObjectID) error {
	return s.userRepo.Delete(ctx, userID)
}
