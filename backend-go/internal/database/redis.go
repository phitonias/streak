package database

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/phitonias/streak/internal/config"
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func ConnectRedis(cfg *config.Config) error {
	// Parse Redis URL
	opts, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		// Fallback for simple URL format
		url := strings.TrimPrefix(cfg.Redis.URL, "redis://")
		opts = &redis.Options{
			Addr: url,
			DB:   0,
		}
	}

	RedisClient = redis.NewClient(opts)

	// Test connection
	ctx := context.Background()
	if err := RedisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	log.Println("✓ Redis connected successfully")
	return nil
}

func CloseRedis() error {
	if RedisClient != nil {
		return RedisClient.Close()
	}
	return nil
}
