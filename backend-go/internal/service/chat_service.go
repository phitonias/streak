package service

import (
	"context"
	"time"

	"github.com/phitonias/streak/internal/database"
	"github.com/phitonias/streak/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ChatService struct{}

func NewChatService() *ChatService {
	return &ChatService{}
}

func (s *ChatService) SaveMessage(message *models.ChatMessage) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.MongoDB.Collection("messages")
	_, err := collection.InsertOne(ctx, message)
	return err
}

func (s *ChatService) GetRecentMessages(streamID string, limit int) ([]models.ChatMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := database.MongoDB.Collection("messages")

	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, bson.M{
		"stream_id":  streamID,
		"is_deleted": false,
	}, opts)

	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.ChatMessage
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, err
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

func (s *ChatService) DeleteMessage(messageID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.MongoDB.Collection("messages")
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": messageID},
		bson.M{"$set": bson.M{"is_deleted": true}},
	)

	return err
}

func (s *ChatService) GetMessageCount(streamID string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.MongoDB.Collection("messages")
	count, err := collection.CountDocuments(ctx, bson.M{
		"stream_id":  streamID,
		"is_deleted": false,
	})

	return count, err
}
