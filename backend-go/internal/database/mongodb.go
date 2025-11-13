package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/phitonias/streak/internal/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoDB *mongo.Database

func ConnectMongoDB(cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to MongoDB
	clientOptions := options.Client().ApplyURI(cfg.MongoDB.URL)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping mongodb: %w", err)
	}

	MongoDB = client.Database(cfg.MongoDB.Database)

	// Create indexes
	if err := createMongoIndexes(ctx); err != nil {
		log.Printf("Warning: failed to create mongodb indexes: %v", err)
	}

	log.Println("✓ MongoDB connected successfully")
	return nil
}

func createMongoIndexes(ctx context.Context) error {
	collection := MongoDB.Collection("messages")

	// Create compound index on stream_id and timestamp
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{
			"stream_id": 1,
			"timestamp": -1,
		},
	})
	if err != nil {
		return err
	}

	// Create index on user_id
	_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{
			"user_id": 1,
		},
	})
	return err
}

func CloseMongoDB() error {
	if MongoDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return MongoDB.Client().Disconnect(ctx)
	}
	return nil
}
