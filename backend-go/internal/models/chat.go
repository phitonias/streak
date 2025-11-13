package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatMessage struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	StreamID  string             `bson:"stream_id" json:"stream_id"`
	UserID    string             `bson:"user_id" json:"user_id"`
	Username  string             `bson:"username" json:"username"`
	Message   string             `bson:"message" json:"message"`
	Timestamp time.Time          `bson:"timestamp" json:"timestamp"`
	IsDeleted bool               `bson:"is_deleted" json:"is_deleted"`
}
