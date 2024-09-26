package index

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
	"tornberg.me/facet-search/pkg/facet"
)

type SortOverride map[uint]float64

func (s *SortOverride) ToString() string {
	ret := ""
	for key, value := range *s {
		ret += fmt.Sprintf("%d:%f,", key, value)
	}
	return ret
}

func (s *SortOverride) FromString(data string) error {
	*s = make(map[uint]float64)
	for _, item := range strings.Split(data, ",") {
		var key uint
		var value float64
		_, err := fmt.Sscanf(item, "%d:%f", &key, &value)
		if err != nil {
			return err
		}
		(*s)[key] = value
	}
	return nil
}

type Sorting struct {
	mu            sync.Mutex
	muOverride    sync.Mutex
	client        *redis.Client
	ctx           context.Context
	fieldOverride *SortOverride
	sortOverrides map[string]*SortOverride
	DefaultSort   *facet.SortIndex
	SortMethods   map[string]*facet.SortIndex
	FieldSort     *facet.SortIndex
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

	pubsub := rdb.Subscribe(ctx, "sortOverrideChange")
	fieldsub := rdb.Subscribe(ctx, "fieldSortOverrideChange")

	go func(ch <-chan *redis.Message) {
		for msg := range ch {
			fmt.Println("Received", msg.Channel, msg.Payload)
			sort_data, err := rdb.Get(ctx, "override_"+msg.Payload).Result()
			if err != nil {
				fmt.Println(err)
				continue
			}
			sort := SortOverride{}
			err = sort.FromString(sort_data)
			if err != nil {
				fmt.Println(err)
				continue
			}
			instance.muOverride.Lock()
			instance.sortOverrides[msg.Pattern] = &sort
			instance.muOverride.Unlock()

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
			sort := SortOverride{}
			err = sort.FromString(sort_data)
			if err != nil {
				fmt.Println(err)
				continue
			}
			instance.muOverride.Lock()
			instance.fieldOverride = &sort
			instance.muOverride.Unlock()
		}
	}(fieldsub.Channel())

	return instance

}

func (s *Sorting) IndexChanged(idx *Index) {
	go func() {
		maps := MakeItemSorting(idx.Items)
		s.mu.Lock()
		defer s.mu.Unlock()
		for key, sort := range maps {
			s.SortMethods[key] = sort
		}
	}()
}

func (s *Sorting) GenerateFieldSort(idx *Index) {
	fieldSort := MakeSortForFields(idx)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FieldSort = &fieldSort
}

func MakeItemSorting(items map[uint]*DataItem) map[string]*facet.SortIndex {
	j := 0.0
	sortMap := MakeSortMap(items, 4, func(value int, item *DataItem) float64 {
		j += 0.000001
		return float64(value) + j
	})
	ret := make(map[string]*facet.SortIndex)
	popularMap := MakePopularSortMap(items)
	ret["popular"] = ToSortIndex(&popularMap, true)
	ret["price"] = ToSortIndex(&sortMap, false)
	ret["price_desc"] = ToSortIndex(&sortMap, true)
	return ret
}

func MakeSortForFields(idx *Index) facet.SortIndex {

	l := len(idx.DecimalFacets) + len(idx.KeyFacets) + len(idx.IntFacets)
	i := 0
	sortIndex := make(facet.SortIndex, l)
	sortMap := make(facet.ByValue, l)

	for _, item := range idx.DecimalFacets {
		if item.HideFacet {
			continue
		}
		sortMap[i] = facet.Lookup{Id: item.Id, Value: item.Priority + float64(item.TotalCount())}
		i++
	}
	for _, item := range idx.KeyFacets {
		if item.HideFacet {
			continue
		}
		sortMap[i] = facet.Lookup{Id: item.Id, Value: item.Priority + float64(item.TotalCount())}
		i++
	}
	for _, item := range idx.IntFacets {
		if item.HideFacet {
			continue
		}
		sortMap[i] = facet.Lookup{Id: item.Id, Value: item.Priority + float64(item.TotalCount())}
		i++
	}
	sortMap = sortMap[:i]
	sort.Sort(sort.Reverse(sortMap))
	for idx, item := range sortMap {
		sortIndex[idx] = item.Id
	}
	return sortIndex
}

func (s *Sorting) Close() {
	s.client.Close()
}

// func (s *Sorting) LoadAll() error {
// 	//rdb := s.client
// 	//ctx := s.ctx
// 	//fieldSortData := rdb.Get(ctx, "fieldSort").Val()
// 	//sortMap := rdb.HGetAll(ctx, "sorts").Val()

// 	// fieldSort := facet.SortIndex{}
// 	// err := fieldSort.FromString(fieldSortData)
// 	// if err != nil {
// 	// 	return err
// 	// }
// 	// s.mu.Lock()
// 	// defer s.mu.Unlock()
// 	// s.FieldSort = &fieldSort
// 	// for key, redisKey := range sortMap {
// 	// 	if key == "price" || key == "price_desc" {
// 	// 		continue
// 	// 	}
// 	// 	sort := facet.SortIndex{}
// 	// 	data := rdb.Get(ctx, redisKey).Val()
// 	// 	if err = sort.FromString(data); err != nil {
// 	// 		return err
// 	// 	}

// 	// 	s.SortMethods[key] = &sort
// 	// }
// 	return nil
// }

func (s *Sorting) AddSortMethodOverride(id string, sort *SortOverride) {
	go func() {
		data := sort.ToString()
		key := "sort_" + id
		s.client.Set(s.ctx, key, data, 0)
		s.client.HSet(s.ctx, "sorts", id, key)
		_, err := s.client.Publish(s.ctx, "sortChange", id).Result()
		if err != nil {
			log.Println("Unable to publish overrides", err)
		}
	}()
	s.muOverride.Lock()
	defer s.muOverride.Unlock()
	s.sortOverrides[id] = sort

}

// func (s *Sorting) SetDefaultSort(defaultSort *facet.SortIndex) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	s.DefaultSort = defaultSort
// }

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

func (s *Sorting) SetFieldSortOverride(sort *SortOverride) {
	go func() {
		data := sort.ToString()
		log.Printf("Setting field sort %d", len(data))
		s.client.Set(s.ctx, "fieldSort", data, 0)
		s.client.Publish(s.ctx, "fieldSortChange", "fieldSort")
	}()
	s.muOverride.Lock()
	defer s.muOverride.Unlock()
	s.fieldOverride = sort
}
