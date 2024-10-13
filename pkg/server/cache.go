package server

import (
	"context"
	"encoding/json"
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
	//memCache map[string]LocalEntry
}

type CacheWriter interface {
	SetRaw(key string, value []byte, expiration time.Duration) error
}

func NewCache(addr, password string, db int) *Cache {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // no password set
		DB:       db,       // use default DB
	})
	return &Cache{Addr: addr, Password: password, DB: db, client: rdb, ctx: ctx} //, memCache: make(map[string]LocalEntry)
}

func (c *Cache) Ping() error {
	_, err := c.client.Ping(c.ctx).Result()
	return err
}

func (c *Cache) Del(key string) error {
	return c.client.Del(c.ctx, key).Err()
}

func (c *Cache) GetRaw(key string) ([]byte, error) {
	return c.client.Get(c.ctx, key).Bytes()
}
func (c *Cache) SetRaw(key string, value []byte, expiration time.Duration) error {
	return c.client.Set(c.ctx, key, value, expiration).Err()
}

func (c *Cache) Get(key string, out any) error {
	// rv := reflect.ValueOf(out)
	// if rv.Kind() != reflect.Pointer || rv.IsNil() {
	// 	return &json.InvalidUnmarshalError{reflect.TypeOf(out)}
	// }
	// local, found := c.memCache[key]
	// if found {
	// 	if local.Expires.Before(time.Now()) {

	// 		rv.Set(local.Data.(reflect.Value))
	// 		//out = local.Data
	// 		return nil
	// 	} else {
	// 		delete(c.memCache, key)
	// 	}

	// }
	data, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(data), out)
	if err != nil {
		return err
	}
	// if out != nil {
	// 	c.memCache[key] = LocalEntry{Expires: time.Now().Add(time.Minute), Data: out}
	// }
	return nil
}

func (c *Cache) Set(key string, value any, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	//c.memCache[key] = LocalEntry{Expires: time.Now().Add(expiration), Data: value}
	return c.client.Set(c.ctx, key, data, expiration).Err()
}

func (c *Cache) Close() {
	c.client.Close()
}
