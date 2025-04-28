package redis

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Ctx context.Context
	Rdb *redis.Client
}

func InitRedisClient() *RedisClient {
	return &RedisClient{
		Ctx: context.Background(),
		Rdb: redis.NewClient(&redis.Options{
			Addr: os.Getenv("REDIS_HOST_NAME") + ":" + os.Getenv("REDIS_PORT"),
			DB:   0,
		}),
	}
}
