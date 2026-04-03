//go:build redis
// +build redis

package memory

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/redis/go-redis/v9"
)

func newRedisStore(dsn string) (Store, error) {
	var opts *redis.Options
	var err error

	if strings.TrimSpace(dsn) == "" {
		dsn = "redis://localhost:6379/0"
	}

	if strings.Contains(dsn, "://") {
		opts, err = redis.ParseURL(dsn)
		if err != nil {
			return nil, err
		}
	} else {
		opts = &redis.Options{Addr: dsn}
	}

	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return &redisStore{client: client}, nil
}

type redisStore struct {
	client *redis.Client
}

func (s *redisStore) key(namespace, key string) string {
	return namespace + ":" + key
}

func (s *redisStore) Get(ctx context.Context, namespace, key string) (any, bool, error) {
	v, err := s.client.Get(ctx, s.key(namespace, key)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}
	var out any
	if err := json.Unmarshal([]byte(v), &out); err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func (s *redisStore) Put(ctx context.Context, namespace, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, s.key(namespace, key), data, 0).Err()
}

func (s *redisStore) List(ctx context.Context, namespace string) ([]string, error) {
	prefix := namespace + ":"
	pattern := prefix + "*"
	keys := make([]string, 0)

	var cursor uint64
	for {
		batch, next, err := s.client.Scan(ctx, cursor, pattern, 200).Result()
		if err != nil {
			return nil, err
		}
		for _, k := range batch {
			keys = append(keys, strings.TrimPrefix(k, prefix))
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

func (s *redisStore) Close() error {
	if s.client == nil {
		return nil
	}
	return s.client.Close()
}
