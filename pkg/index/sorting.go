package index

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

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
			if err.Error() == "EOF" {
				return nil
			}
			return err
		}
		(*s)[key] = value
	}
	return nil
}

type Sorting struct {
	quit             chan struct{}
	idx              *Index
	mu               sync.Mutex
	muOverride       sync.Mutex
	client           *redis.Client
	ctx              context.Context
	fieldOverride    *SortOverride
	popularOverrides *SortOverride
	sortMethods      map[string]*facet.SortIndex
	FieldSort        *facet.SortIndex
	hasItemChanges   bool
}

const POPULAR_SORT = "popular"
const PRICE_SORT = "price"
const PRICE_DESC_SORT = "price_desc"
const REDIS_POPULAR_KEY = "_popular"
const REDIS_POPULAR_CHANGE = "popularChange"
const REDIS_FIELD_KEY = "_field"
const REDIS_FIELD_CHANGE = "fieldChange"

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

		ctx:              ctx,
		client:           rdb,
		sortMethods:      make(map[string]*facet.SortIndex),
		FieldSort:        &facet.SortIndex{},
		popularOverrides: &SortOverride{},
		fieldOverride:    &SortOverride{},
		idx:              nil,
	}

	pubsub := rdb.Subscribe(ctx, REDIS_POPULAR_CHANGE)
	fieldsub := rdb.Subscribe(ctx, REDIS_FIELD_CHANGE)

	popularData, err := rdb.Get(ctx, REDIS_POPULAR_KEY).Result()
	if err == nil {
		sort := SortOverride{}
		err = sort.FromString(popularData)
		if err == nil {
			instance.popularOverrides = &sort
		}
	}

	fieldData, err := rdb.Get(ctx, REDIS_FIELD_KEY).Result()
	if err == nil {
		sort := SortOverride{}
		err = sort.FromString(fieldData)
		if err == nil {
			instance.fieldOverride = &sort
		}
	}
	instance.quit = make(chan struct{})
	ticker := time.NewTicker(15 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Println("tick")
				if instance.hasItemChanges {

					if instance.idx != nil {
						instance.hasItemChanges = false
						instance.muOverride.Lock()
						instance.regeneratePopular(instance.idx)
						maps := MakeItemStaticSorting(instance.idx)
						instance.muOverride.Unlock()
						instance.mu.Lock()
						defer instance.mu.Unlock()
						for key, sort := range maps {
							instance.sortMethods[key] = sort
						}
					}
				}

			case <-instance.quit:
				ticker.Stop()
				return
			}
		}
	}()
	go func(ch <-chan *redis.Message) {
		for msg := range ch {
			fmt.Println("Received popular override change", msg.Channel, msg.Payload)
			sort_data, err := rdb.Get(ctx, REDIS_POPULAR_KEY).Result()
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
			defer instance.muOverride.Unlock()
			instance.popularOverrides = &sort

			instance.hasItemChanges = true
			// if instance.idx != nil {
			// 	instance.regeneratePopular(instance.idx)
			// }
		}
	}(pubsub.Channel())

	go func(ch <-chan *redis.Message) {
		for msg := range ch {
			fmt.Println("Received field sort", msg.Channel, msg.Payload)
			sort_data, err := rdb.Get(ctx, REDIS_FIELD_KEY).Result()
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
			defer instance.muOverride.Unlock()
			instance.fieldOverride = &sort

			if instance.idx != nil {
				instance.GenerateFieldSort(instance.idx)
			}
		}
	}(fieldsub.Channel())

	return instance

}

func (s *Sorting) regeneratePopular(idx *Index) {
	idx.Lock()
	defer idx.Unlock()
	sortMap := makePopularSortMap(idx.Items, *s.popularOverrides)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sortMethods[POPULAR_SORT] = ToSortIndex(&sortMap, true)
}

func (s *Sorting) IndexChanged(idx *Index) {
	s.idx = idx
	s.hasItemChanges = true
}

func (s *Sorting) GenerateFieldSort(idx *Index) {
	idx.Lock()
	defer idx.Unlock()
	fieldSort := MakeSortForFields(idx, *s.fieldOverride)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FieldSort = &fieldSort
}

func MakeItemStaticSorting(idx *Index) map[string]*facet.SortIndex {
	idx.Lock()
	defer idx.Unlock()
	j := 0.0
	sortMap := MakeSortMap(idx.Items, 4, func(value int, item *DataItem) float64 {
		j += 0.000001
		return float64(value) + j
	})
	ret := make(map[string]*facet.SortIndex)

	ret[PRICE_SORT] = ToSortIndex(&sortMap, false)
	ret[PRICE_DESC_SORT] = ToSortIndex(&sortMap, true)
	return ret
}

func MakeSortForFields(idx *Index, overrides SortOverride) facet.SortIndex {
	idx.Lock()
	defer idx.Unlock()
	l := len(idx.DecimalFacets) + len(idx.KeyFacets) + len(idx.IntFacets)
	i := 0
	sortIndex := make(facet.SortIndex, l)
	sortMap := make(facet.ByValue, l)

	for _, item := range idx.DecimalFacets {
		if item.HideFacet {
			continue
		}
		o := 0.0
		manualOverride, hasOverride := overrides[item.Id]
		if hasOverride {
			o = manualOverride
		}
		sortMap[i] = facet.Lookup{Id: item.Id, Value: item.Priority + float64(item.TotalCount()) + o}
		i++
	}
	for _, item := range idx.KeyFacets {
		if item.HideFacet {
			continue
		}
		o := 0.0
		manualOverride, hasOverride := overrides[item.Id]
		if hasOverride {
			o = manualOverride
		}
		sortMap[i] = facet.Lookup{Id: item.Id, Value: item.Priority + float64(item.TotalCount()) + o}
		i++
	}
	for _, item := range idx.IntFacets {
		if item.HideFacet {
			continue
		}
		o := 0.0
		manualOverride, hasOverride := overrides[item.Id]
		if hasOverride {
			o = manualOverride
		}
		sortMap[i] = facet.Lookup{Id: item.Id, Value: item.Priority + float64(item.TotalCount()) + o}
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

func (s *Sorting) AddPopularOverride(sort *SortOverride) {
	data := sort.ToString()
	s.client.Set(s.ctx, REDIS_POPULAR_KEY, data, 0)
	_, err := s.client.Publish(s.ctx, REDIS_POPULAR_CHANGE, "external").Result()
	if err != nil {
		s.muOverride.Lock()
		defer s.muOverride.Unlock()
		s.popularOverrides = sort
		if s.idx != nil {
			s.regeneratePopular(s.idx)
		}
	}
}

func (s *Sorting) GetPopularOverrides() *SortOverride {
	s.muOverride.Lock()
	defer s.muOverride.Unlock()
	return s.popularOverrides
}

func (s *Sorting) GetSort(id string) *facet.SortIndex {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sort, ok := s.sortMethods[id]; ok {
		return sort
	}
	for _, sort := range s.sortMethods {
		return sort
	}
	return &facet.SortIndex{}
}

func (s *Sorting) SetFieldSortOverride(sort *SortOverride) {

	data := sort.ToString()
	log.Printf("Setting field sort %d", len(data))
	s.client.Set(s.ctx, REDIS_FIELD_KEY, data, 0)
	err := s.client.Publish(s.ctx, REDIS_FIELD_CHANGE, "fieldSort")
	if err != nil {
		s.muOverride.Lock()
		defer s.muOverride.Unlock()
		s.fieldOverride = sort
		if s.idx != nil {
			go s.GenerateFieldSort(s.idx)
		}
	}

}
