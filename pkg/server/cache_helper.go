package server

import "time"

type CacheHelper[T any] struct {
	Cache *Cache
}

func NewCacheHelper[T any](cache *Cache) *CacheHelper[T] {
	return &CacheHelper[T]{Cache: cache}
}

func (c *CacheHelper[T]) Handle(key string, out *T, fn func() T, expiration time.Duration) error {
	err := c.Cache.Get(key, out)
	if err != nil {
		*out = fn()
		err = c.Cache.Set(key, out, expiration)
	}
	return err
}
