package redisstore

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct { Client *redis.Client }

func (c *Cache) Get(ctx context.Context, key string) (string,error) { return c.Client.Get(ctx,key).Result() }
func (c *Cache) Set(ctx context.Context,key,value string,ttl time.Duration) error { return c.Client.Set(ctx,key,value,ttl).Err() }
func (c *Cache) Ready(ctx context.Context) error { return c.Client.Ping(ctx).Err() }
