package mongodb

import (
	"context"
	"errors"

	"github.com/ayteuir/backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ReplyRepository struct {
	collection *mongo.Collection
}

func NewReplyRepository(client *Client) *ReplyRepository {
	return &ReplyRepository{
		collection: client.Collection("replies"),
	}
}

func (r *ReplyRepository) Create(ctx context.Context, reply *domain.Reply) error {
	result, err := r.collection.InsertOne(ctx, reply)
	if err != nil {
		return err
	}
	reply.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *ReplyRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Reply, error) {
	var reply domain.Reply
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&reply)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &reply, nil
}

func (r *ReplyRepository) GetByMentionID(ctx context.Context, mentionID primitive.ObjectID) (*domain.Reply, error) {
	var reply domain.Reply
	err := r.collection.FindOne(ctx, bson.M{"mention_id": mentionID}).Decode(&reply)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &reply, nil
}

func (r *ReplyRepository) GetByUserID(ctx context.Context, userID primitive.ObjectID, limit, offset int) ([]*domain.Reply, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var replies []*domain.Reply
	if err := cursor.All(ctx, &replies); err != nil {
		return nil, err
	}
	return replies, nil
}

func (r *ReplyRepository) Update(ctx context.Context, reply *domain.Reply) error {
	result, err := r.collection.ReplaceOne(ctx, bson.M{"_id": reply.ID}, reply)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}
