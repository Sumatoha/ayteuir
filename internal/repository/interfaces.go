package repository

import (
	"context"

	"github.com/ayteuir/backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.User, error)
	GetByThreadsUserID(ctx context.Context, threadsUserID string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

type TemplateRepository interface {
	Create(ctx context.Context, template *domain.Template) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Template, error)
	GetByUserID(ctx context.Context, userID primitive.ObjectID) ([]*domain.Template, error)
	GetByUserIDAndMentionType(ctx context.Context, userID primitive.ObjectID, mentionType domain.MentionType) ([]*domain.Template, error)
	GetActiveByUserIDAndMentionType(ctx context.Context, userID primitive.ObjectID, mentionType domain.MentionType) ([]*domain.Template, error)
	Update(ctx context.Context, template *domain.Template) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

type MentionRepository interface {
	Create(ctx context.Context, mention *domain.Mention) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Mention, error)
	GetByThreadsPostID(ctx context.Context, threadsPostID string) (*domain.Mention, error)
	GetByUserID(ctx context.Context, userID primitive.ObjectID, limit, offset int) ([]*domain.Mention, error)
	GetByUserIDAndStatus(ctx context.Context, userID primitive.ObjectID, status domain.MentionStatus, limit, offset int) ([]*domain.Mention, error)
	GetPendingMentions(ctx context.Context, limit int) ([]*domain.Mention, error)
	Update(ctx context.Context, mention *domain.Mention) error
	CountByUserIDLastHour(ctx context.Context, userID primitive.ObjectID) (int64, error)
}

type ReplyRepository interface {
	Create(ctx context.Context, reply *domain.Reply) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Reply, error)
	GetByMentionID(ctx context.Context, mentionID primitive.ObjectID) (*domain.Reply, error)
	GetByUserID(ctx context.Context, userID primitive.ObjectID, limit, offset int) ([]*domain.Reply, error)
	Update(ctx context.Context, reply *domain.Reply) error
}
