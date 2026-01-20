package mongodb

import (
	"context"
	"time"

	"github.com/ayteuir/backend/internal/config"
	"github.com/ayteuir/backend/internal/pkg/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Client struct {
	client   *mongo.Client
	database *mongo.Database
	cfg      *config.MongoDBConfig
}

func NewClient(cfg *config.MongoDBConfig) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSeconds)*time.Second)
	defer cancel()

	clientOpts := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(100).
		SetMinPoolSize(10).
		SetMaxConnIdleTime(30 * time.Second)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	logger.Info().Msg("Connected to MongoDB")

	return &Client{
		client:   client,
		database: client.Database(cfg.Database),
		cfg:      cfg,
	}, nil
}

func (c *Client) Database() *mongo.Database {
	return c.database
}

func (c *Client) Collection(name string) *mongo.Collection {
	return c.database.Collection(name)
}

func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx, readpref.Primary())
}

func (c *Client) Close(ctx context.Context) error {
	logger.Info().Msg("Closing MongoDB connection")
	return c.client.Disconnect(ctx)
}

func (c *Client) CreateIndexes(ctx context.Context) error {
	indexes := []struct {
		collection string
		models     []mongo.IndexModel
	}{
		{
			collection: "users",
			models: []mongo.IndexModel{
				{
					Keys:    map[string]int{"threads_user_id": 1},
					Options: options.Index().SetUnique(true),
				},
				{
					Keys: map[string]int{"created_at": 1},
				},
			},
		},
		{
			collection: "templates",
			models: []mongo.IndexModel{
				{
					Keys: map[string]int{"user_id": 1, "mention_type": 1},
				},
				{
					Keys: map[string]int{"user_id": 1, "is_active": 1},
				},
			},
		},
		{
			collection: "mentions",
			models: []mongo.IndexModel{
				{
					Keys:    map[string]int{"threads_post_id": 1},
					Options: options.Index().SetUnique(true),
				},
				{
					Keys: map[string]int{"user_id": 1, "status": 1},
				},
				{
					Keys: map[string]int{"webhook_received_at": 1},
				},
			},
		},
		{
			collection: "replies",
			models: []mongo.IndexModel{
				{
					Keys: map[string]int{"user_id": 1, "created_at": -1},
				},
				{
					Keys: map[string]int{"mention_id": 1},
				},
			},
		},
	}

	for _, idx := range indexes {
		coll := c.Collection(idx.collection)
		_, err := coll.Indexes().CreateMany(ctx, idx.models)
		if err != nil {
			logger.Error().Err(err).Str("collection", idx.collection).Msg("Failed to create indexes")
			return err
		}
		logger.Info().Str("collection", idx.collection).Msg("Created indexes")
	}

	return nil
}
