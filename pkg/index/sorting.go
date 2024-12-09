package index

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/matst80/slask-finder/pkg/types"
	"github.com/redis/go-redis/v9"
)

type Sorting struct {
	quit             chan struct{}
	idx              *Index
	mu               sync.RWMutex
	muStaticPos      sync.RWMutex
	muOverride       sync.RWMutex
	client           *redis.Client
	fieldOverride    *SortOverride
	popularOverrides *SortOverride
	sortMethods      map[string]*types.SortIndex
	staticPositions  *StaticPositions
	FieldSort        *types.SortIndex
	hasItemChanges   bool
}

const POPULAR_SORT = "popular"
const PRICE_SORT = "price"
const UPDATED_SORT = "updated"
const CREATED_SORT = "created"
const UPDATED_DESC_SORT = "updated_desc"
const CREATED_DESC_SORT = "created_desc"

const PRICE_DESC_SORT = "price_desc"
const REDIS_POPULAR_KEY = "_popular"
const REDIS_POPULAR_CHANGE = "popularChange"
const REDIS_FIELD_KEY = "_field"
const REDIS_FIELD_CHANGE = "fieldChange"
const REDIS_STATIC_KEY = "_staticPositions"
const REDIS_STATIC_CHANGE = "staticPositionsChange"

func NewSorting(addr, password string, db int) *Sorting {

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	instance := &Sorting{

		client:           rdb,
		sortMethods:      make(map[string]*types.SortIndex),
		FieldSort:        &types.SortIndex{},
		popularOverrides: &SortOverride{},
		fieldOverride:    &SortOverride{},
		staticPositions:  &StaticPositions{},
		idx:              nil,
	}

	return instance

}

func (s *Sorting) StartListeningForChanges() {
	rdb := s.client
	ctx := context.Background()
	pubsub := rdb.Subscribe(ctx, REDIS_POPULAR_CHANGE)
	fieldsub := rdb.Subscribe(ctx, REDIS_FIELD_CHANGE)
	staticsub := rdb.Subscribe(ctx, REDIS_STATIC_CHANGE)

	go func(ch <-chan *redis.Message) {
		for range ch {
			// fmt.Println("Received popular override change", msg.Channel, msg.Payload)
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
			s.muOverride.Lock()
			s.popularOverrides = &sort
			s.muOverride.Unlock()

			s.hasItemChanges = true

		}
	}(pubsub.Channel())

	go func(ch <-chan *redis.Message) {
		for range ch {
			//fmt.Println("Received static positions change", msg.Channel, msg.Payload)
			sort_data, err := rdb.Get(ctx, REDIS_STATIC_KEY).Result()
			if err != nil {
				fmt.Println(err)
				continue
			}
			sort := StaticPositions{}
			err = sort.FromString(sort_data)
			if err != nil {
				fmt.Println(err)
				continue
			}
			s.setStaticPositions(sort)

		}
	}(staticsub.Channel())

	go func(ch <-chan *redis.Message) {
		for range ch {
			//fmt.Println("Received field sort", msg.Channel, msg.Payload)
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
			s.muOverride.Lock()
			s.fieldOverride = &sort
			s.muOverride.Unlock()

			if s.idx != nil {
				s.makeItemSortMaps()
			}
		}
	}(fieldsub.Channel())
}

func (s *Sorting) IndexChanged(idx *Index) {
	s.idx = idx
	s.hasItemChanges = true
}

func (s *Sorting) GetStaticPositions() StaticPositions {
	s.muStaticPos.RLock()
	defer s.muStaticPos.RUnlock()
	return *s.staticPositions
}

func (s *Sorting) SetStaticPositions(positions StaticPositions) error {
	s.setStaticPositions(positions)
	ctx := context.Background()
	data := positions.ToString()
	s.client.Set(ctx, REDIS_STATIC_KEY, data, 0)
	_, err := s.client.Publish(ctx, REDIS_STATIC_CHANGE, "external").Result()
	return err
}

func (s *Sorting) setStaticPositions(positions StaticPositions) {
	s.muStaticPos.Lock()
	defer s.muStaticPos.Unlock()
	s.staticPositions = &positions
}

func (s *Sorting) InitializeWithIndex(idx *Index) {
	ctx := context.Background()
	popularData, err := s.client.Get(ctx, REDIS_POPULAR_KEY).Result()
	s.idx = idx
	if err == nil {
		sort := SortOverride{}
		err = sort.FromString(popularData)
		if err == nil {
			s.muOverride.Lock()
			s.popularOverrides = &sort
			s.muOverride.Unlock()
		}
	}

	fieldData, err := s.client.Get(ctx, REDIS_FIELD_KEY).Result()
	if err == nil {
		sort := SortOverride{}
		err = sort.FromString(fieldData)
		if err == nil {
			s.muOverride.Lock()
			s.fieldOverride = &sort
			s.muOverride.Unlock()
		}
	}

	staticData, err := s.client.Get(ctx, REDIS_STATIC_KEY).Result()
	if err == nil {
		sort := StaticPositions{}
		err = sort.FromString(staticData)
		if err == nil {
			s.setStaticPositions(sort)
		}
	}
	s.makeFieldSort(idx, *s.fieldOverride)
	s.makeItemSortMaps()
	s.hasItemChanges = false
	log.Println("Sorting initialized")
	s.quit = make(chan struct{})
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:

				if s.hasItemChanges {
					log.Println("items changed, updating sort maps")
					if s.idx != nil {
						s.hasItemChanges = false
						s.makeItemSortMaps()
					}
				}

			case <-s.quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func getFieldLookupValue(field types.BaseField, overrideValue float64) types.Lookup {
	if field.HideFacet {
		return types.Lookup{Id: field.Id, Value: 0}
	}

	return types.Lookup{Id: field.Id, Value: field.Priority + overrideValue}
}

func (s *Sorting) makeFieldSort(idx *Index, overrides SortOverride) {
	idx.Lock()
	defer idx.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	l := len(idx.Facets)
	i := 0
	sortIndex := make(types.SortIndex, l)
	sortMap := make(types.ByValue, l)
	var base *types.BaseField
	for _, item := range idx.Facets {
		base = item.GetBaseField()
		if base.HideFacet {
			continue
		}
		sortMap[i] = getFieldLookupValue(*base, overrides[base.Id])
		i++
	}

	sortMap = sortMap[:i]
	sort.Sort(sort.Reverse(sortMap))
	for idx, item := range sortMap {
		sortIndex[idx] = item.Id
	}

	s.FieldSort = ToSortIndex(&sortMap, true)
}

func (s *Sorting) Close() {
	s.client.Close()
}

func (s *Sorting) AddPopularOverride(sort *SortOverride) {
	data := sort.ToString()
	ctx := context.Background()
	s.client.Set(ctx, REDIS_POPULAR_KEY, data, 0)
	_, err := s.client.Publish(ctx, REDIS_POPULAR_CHANGE, "external").Result()
	if err != nil {
		s.muOverride.Lock()
		defer s.muOverride.Unlock()
		s.popularOverrides = sort
		s.hasItemChanges = true
	}
}

func (s *Sorting) GetPopularOverrides() *SortOverride {
	s.muOverride.RLock()
	defer s.muOverride.RUnlock()
	return s.popularOverrides
}

func (s *Sorting) GetSorting(id string, sortChan chan *types.SortIndex) {
	sortChan <- s.GetSort(id)
}

func (s *Sorting) GetSort(id string) *types.SortIndex {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if sort, ok := s.sortMethods[id]; ok {
		return sort
	}
	for _, sort := range s.sortMethods {
		return sort
	}
	return &types.SortIndex{}
}

func (s *Sorting) makeItemSortMaps() {
	s.idx.Lock()
	defer s.idx.Unlock()

	s.muOverride.Lock()
	defer s.muOverride.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	overrides := *s.popularOverrides

	l := len(s.idx.Items)
	j := 0.0
	now := time.Now()
	ts := now.Unix() / 1000
	popularMap := make(types.ByValue, l)
	priceMap := make(types.ByValue, l)
	updatedMap := make(types.ByValue, l)
	createdMap := make(types.ByValue, l)
	popularSearchMap := make(map[uint]float64)
	i := 0
	var item types.Item
	var itm *types.Item
	var id uint
	for id, itm = range s.idx.Items {
		item = *itm
		j += 0.0000000000001
		popular := item.GetPopularity() + (overrides[item.GetId()] * 1000)

		partPopular := popular / 1000.0
		if item.GetLastUpdated() == 0 {
			updatedMap[i] = types.Lookup{Id: id, Value: j}
		} else {
			updatedMap[i] = types.Lookup{Id: id, Value: float64(ts-item.GetLastUpdated()/1000) + j}
		}
		if item.GetCreated() == 0 {
			createdMap[i] = types.Lookup{Id: id, Value: partPopular + j}
		} else {
			createdMap[i] = types.Lookup{Id: id, Value: partPopular + float64(ts-item.GetCreated()/1000) + j}
		}

		priceMap[i] = types.Lookup{Id: id, Value: float64(item.GetPrice()) + j}
		popularMap[i] = types.Lookup{Id: id, Value: popular + j}
		popularSearchMap[id] = popular / 1000.0
		i++
	}
	if s.idx != nil {
		s.idx.SetBaseSortMap(popularSearchMap)
	}
	s.sortMethods[POPULAR_SORT] = ToSortIndex(&popularMap, true)
	s.sortMethods[PRICE_SORT] = ToSortIndex(&priceMap, false)
	s.sortMethods[PRICE_DESC_SORT] = ToSortIndex(&priceMap, true)
	s.sortMethods[UPDATED_SORT] = ToSortIndex(&updatedMap, false)
	s.sortMethods[UPDATED_DESC_SORT] = ToSortIndex(&updatedMap, true)
	s.sortMethods[CREATED_SORT] = ToSortIndex(&createdMap, false)
	s.sortMethods[CREATED_DESC_SORT] = ToSortIndex(&createdMap, true)

}

func (s *Sorting) SetFieldSortOverride(sort *SortOverride) {
	ctx := context.Background()
	data := sort.ToString()
	log.Printf("Setting field sort %d", len(data))
	s.client.Set(ctx, REDIS_FIELD_KEY, data, 0)
	err := s.client.Publish(ctx, REDIS_FIELD_CHANGE, "fieldSort")
	if err != nil {
		if s.idx != nil {
			go s.makeFieldSort(s.idx, *sort)
		}
		s.muOverride.Lock()
		defer s.muOverride.Unlock()
		s.fieldOverride = sort
	}

}
