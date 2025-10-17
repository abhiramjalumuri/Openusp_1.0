package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"openusp/pkg/config"
)

// Client wraps Redis client with OpenUSP-specific operations
type Client struct {
	client *redis.Client
	ctx    context.Context
}

// NewClient creates a new Redis client from configuration
func NewClient(cfg *config.RedisConfig) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("redis config is nil")
	}

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	
	log.Printf("🔌 Connecting to Redis at %s (DB: %d)...", addr, cfg.DB)

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		PoolTimeout:  4 * time.Second,
		MaxRetries:   3,
	})

	ctx := context.Background()

	// Test connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("✅ Redis connection established successfully")

	return &Client{
		client: rdb,
		ctx:    ctx,
	}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Ping checks if Redis is responsive
func (c *Client) Ping() error {
	return c.client.Ping(c.ctx).Err()
}

// Set stores a key-value pair with optional expiration
func (c *Client) Set(key string, value interface{}, expiration time.Duration) error {
	// Convert value to JSON
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return c.client.Set(c.ctx, key, jsonData, expiration).Err()
}

// Get retrieves a value and unmarshals it into the provided interface
func (c *Client) Get(key string, dest interface{}) error {
	val, err := c.client.Get(c.ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("key does not exist: %s", key)
	}
	if err != nil {
		return fmt.Errorf("failed to get key: %w", err)
	}

	// Unmarshal JSON into destination
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

// Exists checks if a key exists
func (c *Client) Exists(key string) (bool, error) {
	result, err := c.client.Exists(c.ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// Delete removes one or more keys
func (c *Client) Delete(keys ...string) error {
	return c.client.Del(c.ctx, keys...).Err()
}

// Expire sets expiration time on a key
func (c *Client) Expire(key string, expiration time.Duration) error {
	return c.client.Expire(c.ctx, key, expiration).Err()
}

// Keys returns all keys matching a pattern
func (c *Client) Keys(pattern string) ([]string, error) {
	return c.client.Keys(c.ctx, pattern).Result()
}

// HSet sets a field in a hash
func (c *Client) HSet(key string, field string, value interface{}) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return c.client.HSet(c.ctx, key, field, jsonData).Err()
}

// HGet gets a field from a hash
func (c *Client) HGet(key string, field string, dest interface{}) error {
	val, err := c.client.HGet(c.ctx, key, field).Result()
	if err == redis.Nil {
		return fmt.Errorf("field does not exist: %s", field)
	}
	if err != nil {
		return fmt.Errorf("failed to get hash field: %w", err)
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

// HGetAll gets all fields from a hash
func (c *Client) HGetAll(key string) (map[string]string, error) {
	return c.client.HGetAll(c.ctx, key).Result()
}

// HDel deletes fields from a hash
func (c *Client) HDel(key string, fields ...string) error {
	return c.client.HDel(c.ctx, key, fields...).Err()
}

// Incr increments the value of a key
func (c *Client) Incr(key string) (int64, error) {
	return c.client.Incr(c.ctx, key).Result()
}

// Decr decrements the value of a key
func (c *Client) Decr(key string) (int64, error) {
	return c.client.Decr(c.ctx, key).Result()
}

// GetClient returns the underlying Redis client for advanced operations
func (c *Client) GetClient() *redis.Client {
	return c.client
}

// GetContext returns the context used for Redis operations
func (c *Client) GetContext() context.Context {
	return c.ctx
}

// FlushDB clears all keys in the current database (use with caution!)
func (c *Client) FlushDB() error {
	return c.client.FlushDB(c.ctx).Err()
}

// Info returns Redis server information
func (c *Client) Info() (string, error) {
	return c.client.Info(c.ctx).Result()
}
