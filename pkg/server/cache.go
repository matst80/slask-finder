package server

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/redis/go-redis/v9"
)

type LocalEntry struct {
	Expires time.Time
	Data    any
}

type Cache struct {
	Addr     string
	Password string
	DB       int
	client   *redis.Client
	ctx      context.Context
	memCache map[string]LocalEntry
}

func NewCache(addr, password string, db int) *Cache {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // no password set
		DB:       db,       // use default DB
	})
	return &Cache{Addr: addr, Password: password, DB: db, client: rdb, ctx: ctx, memCache: make(map[string]LocalEntry)}
}

func (c *Cache) Get(key string, out any) error {
	local, found := c.memCache[key]
	if found {
		if local.Expires.Before(time.Now()) {
			rv := reflect.ValueOf(out)
			rv.Set(local.Data.(reflect.Value))
			//out = local.Data
			return nil
		} else {
			delete(c.memCache, key)
		}

	}
	data, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(data), out)
	if err != nil {
		return err
	}
	c.memCache[key] = LocalEntry{Expires: time.Now().Add(time.Minute), Data: out}
	return nil
}

func (c *Cache) Set(key string, value any, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.memCache[key] = LocalEntry{Expires: time.Now().Add(expiration), Data: value}
	return c.client.Set(c.ctx, key, data, expiration).Err()
}

func (c *Cache) Close() {
	c.client.Close()
}

func (c *Cache) CacheKey(key string, instance *any, fn func() *any, expiration time.Duration) error {
	err := c.Get(key, instance)
	if err != nil {
		*instance = fn()
		return c.Set(key, *instance, expiration)
	}
	return nil
}
