package rediscache

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/edgestore/edgestore/internal/errors"
	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client *redis.Client
}

// New returns a new Redis cache store.
func New(client *redis.Client) *Redis {
	return &Redis{client}
}

func (c *Redis) Get(ctx context.Context, key string, value interface{}) error {
	const op errors.Op = "persistence/Redis.Get"
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		switch {
		case err == redis.Nil:
			return errors.E(op, errors.NotFound)
		default:
			return errors.E(op, err, errors.Internal)
		}
	}

	if err := json.Unmarshal(bytes.NewBufferString(data).Bytes(), value); err != nil {
		return errors.E(op, err, errors.Internal)
	}

	return nil
}

func (c *Redis) Set(ctx context.Context, key string, value interface{}, expires time.Duration) error {
	const op errors.Op = "persistence/Redis.Set"

	if _, err := c.client.Set(ctx, key, value, expires).Result(); err != nil {
		return errors.E(op, err, errors.Internal)
	}

	return nil
}

func (c *Redis) Delete(ctx context.Context, key string) error {
	const op errors.Op = "persistence/Redis.Delete"

	if _, err := c.client.Del(ctx, key).Result(); err != nil {
		return errors.E(op, err, errors.Internal)
	}

	return nil
}

// Flush deletes all items from client asynchronous
func (c *Redis) Flush(ctx context.Context) error {
	const op errors.Op = "persistence/Redis.Flush"
	if _, err := c.client.FlushAllAsync(ctx).Result(); err != nil {
		return errors.E(op, errors.Internal, err)
	}

	return nil
}

func (c *Redis) Run() error {
	if _, err := c.client.Ping(context.Background()).Result(); err != nil {
		return err
	}

	return nil
}

func (c *Redis) Shutdown() error {
	if _, err := c.client.Shutdown(context.Background()).Result(); err != nil {
		return err
	}

	return nil
}
