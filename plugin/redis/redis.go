package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vultisig/verifier/plugin/config"
	"github.com/vultisig/vultiserver/contexthelper"
)

type Redis struct {
	cfg    config.Redis
	client *redis.Client
}

func NewRedis(cfg config.Redis) (*Redis, error) {
	opts, err := cfg.GetRedisOptions()
	if err != nil {
		return nil, fmt.Errorf("failed to get redis opts: %w", err)
	}

	client := redis.NewClient(opts)
	status := client.Ping(context.Background())
	if status.Err() != nil {
		return nil, status.Err()
	}
	return &Redis{
		cfg:    cfg,
		client: client,
	}, nil
}

func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return "", err
	}
	return r.client.Get(ctx, key).Result()
}
func (r *Redis) Set(ctx context.Context, key string, value string, expiry time.Duration) error {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return err
	}
	return r.client.Set(ctx, key, value, expiry).Err()
}
func (r *Redis) Expire(ctx context.Context, key string, expiry time.Duration) error {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return err
	}
	return r.client.Expire(ctx, key, expiry).Err()
}
func (r *Redis) Delete(ctx context.Context, key string) error {
	if err := contexthelper.CheckCancellation(ctx); err != nil {
		return err
	}
	return r.client.Del(ctx, key).Err()
}
func (r *Redis) Close() error {
	return r.client.Close()
}
