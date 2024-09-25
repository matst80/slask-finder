package server

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/redis/go-redis/v9"
	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
)

type Sorting struct {
	mu          sync.Mutex
	client      *redis.Client
	ctx         context.Context
	DefaultSort *facet.SortIndex
	SortMethods map[string]*facet.SortIndex
	FieldSort   *facet.SortIndex
}

func NewSorting(addr, password string, db int) *Sorting {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// rdbPubSub := redis.NewClient(&redis.Options{
	// 	Addr:     addr,
	// 	Password: password,
	// 	DB:       db,
	// })

	instance := &Sorting{

		ctx:         ctx,
		client:      rdb,
		DefaultSort: &facet.SortIndex{},
		SortMethods: make(map[string]*facet.SortIndex),
		FieldSort:   &facet.SortIndex{},
	}

	pubsub := rdb.Subscribe(ctx, "sortChange")
	fieldsub := rdb.Subscribe(ctx, "fieldSortChange")

	go func(ch <-chan *redis.Message) {
		for msg := range ch {
			fmt.Println("Received", msg.Channel, msg.Payload)
			sort_data, err := rdb.Get(ctx, "sort_"+msg.Payload).Result()
			if err != nil {
				fmt.Println(err)
				continue
			}
			sort := facet.SortIndex{}
			err = sort.FromString(sort_data)
			if err != nil {
				fmt.Println(err)
				continue
			}
			instance.mu.Lock()
			instance.SortMethods[msg.Payload] = &sort
			instance.mu.Unlock()

		}
	}(pubsub.Channel())

	go func(ch <-chan *redis.Message) {
		for msg := range ch {
			fmt.Println("Received field sort", msg.Channel, msg.Payload)
			sort_data, err := rdb.Get(ctx, "fieldSort").Result()
			if err != nil {
				fmt.Println(err)
				continue
			}
			sort := facet.SortIndex{}
			err = sort.FromString(sort_data)
			if err != nil {
				fmt.Println(err)
				continue
			}
			instance.mu.Lock()
			instance.FieldSort = &sort
			instance.mu.Unlock()
		}
	}(fieldsub.Channel())

	return instance

}

func (s *Sorting) ItemAdded(item *index.DataItem) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, sort := range s.SortMethods {
		sort.Add(item.Id)
		s.AddSortMethod(key, sort)
	}
}

func (s *Sorting) ItemDeleted(itemId uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, sort := range s.SortMethods {
		sort.Remove(itemId)
		s.AddSortMethod(key, sort)
	}
}

func (s *Sorting) Close() {
	s.client.Close()
}

func (s *Sorting) LoadAll() error {
	rdb := s.client
	ctx := s.ctx
	fieldSortData := rdb.Get(ctx, "fieldSort").Val()
	sortMap := rdb.HGetAll(ctx, "sorts").Val()

	fieldSort := facet.SortIndex{}
	err := fieldSort.FromString(fieldSortData)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FieldSort = &fieldSort
	for key, redisKey := range sortMap {
		sort := facet.SortIndex{}
		data := rdb.Get(ctx, redisKey).Val()
		if err = sort.FromString(data); err != nil {
			return err
		}

		s.SortMethods[key] = &sort
	}
	return nil
}

func (s *Sorting) AddSortMethod(id string, sort *facet.SortIndex) error {
	data := sort.ToString()
	key := "sort_" + id
	s.client.Set(s.ctx, key, data, 0)
	s.client.HSet(s.ctx, "sorts", id, key)
	_, err := s.client.Publish(s.ctx, "sortChange", id).Result()
	return err

}

func (s *Sorting) SetDefaultSort(defaultSort *facet.SortIndex) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DefaultSort = defaultSort
}

func (s *Sorting) GetSort(id string) *facet.SortIndex {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sort, ok := s.SortMethods[id]; ok {
		return sort
	}
	if s.DefaultSort != nil {
		return s.DefaultSort
	}
	for _, sort := range s.SortMethods {
		s.DefaultSort = sort
		return sort
	}
	return &facet.SortIndex{}
}

func (s *Sorting) SetFieldSort(sort *facet.SortIndex) {

	data := sort.ToString()
	log.Printf("Setting field sort %d", len(data))
	s.client.Set(s.ctx, "fieldSort", data, 0)
	s.client.Publish(s.ctx, "fieldSortChange", "fieldSort")
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FieldSort = sort
}
