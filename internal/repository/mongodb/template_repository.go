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

type TemplateRepository struct {
	collection *mongo.Collection
}

func NewTemplateRepository(client *Client) *TemplateRepository {
	return &TemplateRepository{
		collection: client.Collection("templates"),
	}
}

func (r *TemplateRepository) Create(ctx context.Context, template *domain.Template) error {
	result, err := r.collection.InsertOne(ctx, template)
	if err != nil {
		return err
	}
	template.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *TemplateRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Template, error) {
	var template domain.Template
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&template)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &template, nil
}

func (r *TemplateRepository) GetByUserID(ctx context.Context, userID primitive.ObjectID) ([]*domain.Template, error) {
	opts := options.Find().SetSort(bson.D{{Key: "priority", Value: 1}, {Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var templates []*domain.Template
	if err := cursor.All(ctx, &templates); err != nil {
		return nil, err
	}
	return templates, nil
}

func (r *TemplateRepository) GetByUserIDAndMentionType(ctx context.Context, userID primitive.ObjectID, mentionType domain.MentionType) ([]*domain.Template, error) {
	filter := bson.M{
		"user_id":      userID,
		"mention_type": mentionType,
	}
	opts := options.Find().SetSort(bson.D{{Key: "priority", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var templates []*domain.Template
	if err := cursor.All(ctx, &templates); err != nil {
		return nil, err
	}
	return templates, nil
}

func (r *TemplateRepository) GetActiveByUserIDAndMentionType(ctx context.Context, userID primitive.ObjectID, mentionType domain.MentionType) ([]*domain.Template, error) {
	filter := bson.M{
		"user_id":      userID,
		"mention_type": mentionType,
		"is_active":    true,
	}
	opts := options.Find().SetSort(bson.D{{Key: "priority", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var templates []*domain.Template
	if err := cursor.All(ctx, &templates); err != nil {
		return nil, err
	}
	return templates, nil
}

func (r *TemplateRepository) Update(ctx context.Context, template *domain.Template) error {
	result, err := r.collection.ReplaceOne(ctx, bson.M{"_id": template.ID}, template)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *TemplateRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}
