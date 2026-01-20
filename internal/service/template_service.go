package service

import (
	"context"

	"github.com/ayteuir/backend/internal/domain"
	"github.com/ayteuir/backend/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TemplateService struct {
	templateRepo repository.TemplateRepository
}

func NewTemplateService(templateRepo repository.TemplateRepository) *TemplateService {
	return &TemplateService{
		templateRepo: templateRepo,
	}
}

func (s *TemplateService) Create(ctx context.Context, userID primitive.ObjectID, name string, mentionType domain.MentionType, content string) (*domain.Template, error) {
	template := domain.NewTemplate(userID, name, mentionType, content)
	if err := s.templateRepo.Create(ctx, template); err != nil {
		return nil, err
	}
	return template, nil
}

func (s *TemplateService) GetByID(ctx context.Context, userID, templateID primitive.ObjectID) (*domain.Template, error) {
	template, err := s.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	if template.UserID != userID {
		return nil, domain.ErrForbidden
	}

	return template, nil
}

func (s *TemplateService) GetAll(ctx context.Context, userID primitive.ObjectID) ([]*domain.Template, error) {
	return s.templateRepo.GetByUserID(ctx, userID)
}

func (s *TemplateService) Update(ctx context.Context, userID, templateID primitive.ObjectID, name, content string, isActive bool, priority int) (*domain.Template, error) {
	template, err := s.GetByID(ctx, userID, templateID)
	if err != nil {
		return nil, err
	}

	template.Update(name, content, isActive, priority)

	if err := s.templateRepo.Update(ctx, template); err != nil {
		return nil, err
	}

	return template, nil
}

func (s *TemplateService) Delete(ctx context.Context, userID, templateID primitive.ObjectID) error {
	template, err := s.GetByID(ctx, userID, templateID)
	if err != nil {
		return err
	}

	return s.templateRepo.Delete(ctx, template.ID)
}
