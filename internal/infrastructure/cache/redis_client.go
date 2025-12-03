package cache

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/redis/go-redis/v9"
)

// RedisConfig holds the Redis configuration.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

var (
	redisClient *redis.Client
	redisOnce   sync.Once
)

// GetRedisClient initializes and returns the Redis client.
func GetRedisClient(ctx context.Context, config *RedisConfig) (*redis.Client, error) {
	var err error
	redisOnce.Do(func() {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     config.Addr,
			Password: config.Password,
			DB:       config.DB,
			// PoolSize: 10, // You can configure pool size here
		})

		// Ping to check connection
		_, err = redisClient.Ping(ctx).Result()
		if err != nil {
			err = fmt.Errorf("failed to ping Redis: %w", err)
			redisClient = nil // Clear client if ping fails
		}
	})

	if err != nil {
		return nil, err
	}
	return redisClient, nil
}

// CloseRedisClient closes the Redis client connection.
func CloseRedisClient() {
	if redisClient != nil {
		_ = redisClient.Close()
	}
}

// GetRedisConfigFromEnv loads Redis configuration from environment variables.
func GetRedisConfigFromEnv() *RedisConfig {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	password := os.Getenv("REDIS_PASSWORD")
	dbStr := os.Getenv("REDIS_DB")
	db := 0
	if dbStr != "" {
		if i, err := strconv.Atoi(dbStr); err == nil {
			db = i
		}
	}

	return &RedisConfig{
		Addr:     addr,
		Password: password,
		DB:       db,
	}
}