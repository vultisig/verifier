package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vultisig/verifier/plugin/config"
	"github.com/vultisig/vultiserver/contexthelper"
)

type RedisStorage struct {
	cfg    config.Redis
	client *redis.Client
}

func NewRedisStorage(cfg config.Redis) (*RedisStorage, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Host + ":" + cfg.Port,
		Username: cfg.User,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	status := client.Ping(context.Background())
	if status.Err() != nil {
		return nil, status.Err()
	}
	return &RedisStorage{
		client: client,
	}, nil
}

func (r *RedisStorage) Get(ctx context.Context, key string) (string, error) {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return "", err
	}
	return r.client.Get(ctx, key).Result()
}

func (r *RedisStorage) Set(ctx context.Context, key string, value string, expiry time.Duration) error {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return err
	}
	return r.client.Set(ctx, key, value, expiry).Err()
}

func (r *RedisStorage) Exists(ctx context.Context, key string) (bool, error) {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return false, err
	}
	val, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}
	return val > 0, nil
}

func (r *RedisStorage) Expire(ctx context.Context, key string, expiry time.Duration) error {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return err
	}
	return r.client.Expire(ctx, key, expiry).Err()
}

func (r *RedisStorage) Delete(ctx context.Context, key string) error {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return err
	}
	return r.client.Del(ctx, key).Err()
}

func (r *RedisStorage) Close() error {
	return r.client.Close()
}

// CheckNonceExists checks if a nonce has been used by the same public key
func (r *RedisStorage) CheckNonceExists(ctx context.Context, nonce string, publicKey string) (bool, error) {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return false, err
	}
	key := fmt.Sprintf("%s:%s", publicKey, nonce)
	return r.Exists(ctx, key)
}

// StoreNonce stores a nonce in Redis with automatic expiration
func (r *RedisStorage) StoreNonce(ctx context.Context, nonce string, publicKey string, expiryTime time.Time) error {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return err
	}
	if expiryTime.Before(time.Now()) {
		return fmt.Errorf("expiry time cannot be in the past")
	}
	key := fmt.Sprintf("%s:%s", publicKey, nonce)
	expiryDuration := time.Until(expiryTime)
	return r.Set(ctx, key, "1", expiryDuration)
}
