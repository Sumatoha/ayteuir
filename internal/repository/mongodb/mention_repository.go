package mongodb

import (
	"context"
	"errors"
	"time"

	"github.com/ayteuir/backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MentionRepository struct {
	collection *mongo.Collection
}

func NewMentionRepository(client *Client) *MentionRepository {
	return &MentionRepository{
		collection: client.Collection("mentions"),
	}
}

func (r *MentionRepository) Create(ctx context.Context, mention *domain.Mention) error {
	result, err := r.collection.InsertOne(ctx, mention)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.ErrDuplicateEntry
		}
		return err
	}
	mention.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *MentionRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Mention, error) {
	var mention domain.Mention
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&mention)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &mention, nil
}

func (r *MentionRepository) GetByThreadsPostID(ctx context.Context, threadsPostID string) (*domain.Mention, error) {
	var mention domain.Mention
	err := r.collection.FindOne(ctx, bson.M{"threads_post_id": threadsPostID}).Decode(&mention)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &mention, nil
}

func (r *MentionRepository) GetByUserID(ctx context.Context, userID primitive.ObjectID, limit, offset int) ([]*domain.Mention, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var mentions []*domain.Mention
	if err := cursor.All(ctx, &mentions); err != nil {
		return nil, err
	}
	return mentions, nil
}

func (r *MentionRepository) GetByUserIDAndStatus(ctx context.Context, userID primitive.ObjectID, status domain.MentionStatus, limit, offset int) ([]*domain.Mention, error) {
	filter := bson.M{
		"user_id": userID,
		"status":  status,
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var mentions []*domain.Mention
	if err := cursor.All(ctx, &mentions); err != nil {
		return nil, err
	}
	return mentions, nil
}

func (r *MentionRepository) GetPendingMentions(ctx context.Context, limit int) ([]*domain.Mention, error) {
	filter := bson.M{"status": domain.MentionStatusPending}
	opts := options.Find().
		SetSort(bson.D{{Key: "webhook_received_at", Value: 1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var mentions []*domain.Mention
	if err := cursor.All(ctx, &mentions); err != nil {
		return nil, err
	}
	return mentions, nil
}

func (r *MentionRepository) Update(ctx context.Context, mention *domain.Mention) error {
	result, err := r.collection.ReplaceOne(ctx, bson.M{"_id": mention.ID}, mention)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *MentionRepository) CountByUserIDLastHour(ctx context.Context, userID primitive.ObjectID) (int64, error) {
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	filter := bson.M{
		"user_id": userID,
		"status":  domain.MentionStatusReplied,
		"processed_at": bson.M{
			"$gte": oneHourAgo,
		},
	}
	return r.collection.CountDocuments(ctx, filter)
}
