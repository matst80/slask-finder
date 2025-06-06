package index

import (
	"cmp"
	"context"
	"fmt"
	"iter"
	"log"
	"slices"
	"strconv"
	"strings"
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
	popularMap       *SortOverride
	fieldMap         *SortOverride
	sessionOverrides map[uint]*SortOverride
	//sessionFieldOverrides map[uint]*SortOverride
	sortMethods map[string]*types.ByValue
	//staticPositions       *StaticPositions
	FieldSort *types.ByValue
	//popularityRules       *types.ItemPopularityRules
	hasItemChanges bool
}

const POPULAR_SORT = "popular"
const PRICE_SORT = "price"
const UPDATED_SORT = "updated"
const CREATED_SORT = "created"

// const UPDATED_DESC_SORT = "updated_desc"
// const CREATED_DESC_SORT = "created_desc"

const PRICE_DESC_SORT = "price_desc"
const REDIS_POPULAR_KEY = "_popular"
const REDIS_POPULAR_CHANGE = "popularChange"
const REDIS_FIELD_KEY = "_field"
const REDIS_FIELD_CHANGE = "fieldChange"
const REDIS_STATIC_KEY = "_staticPositions"
const REDIS_STATIC_CHANGE = "staticPositionsChange"
const REDIS_SESSION_POPULAR_CHANGE = "sessionChange"
const REDIS_SESSION_FIELD_CHANGE = "sessionFieldChange"

func NewSorting(addr, password string, db int) *Sorting {

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	instance := &Sorting{

		client:           rdb,
		sortMethods:      make(map[string]*types.ByValue),
		FieldSort:        &types.ByValue{},
		popularOverrides: &SortOverride{},
		fieldMap:         &SortOverride{},
		sessionOverrides: make(map[uint]*SortOverride),
		//sessionFieldOverrides: make(map[uint]*SortOverride),
		fieldOverride: &SortOverride{},
		//staticPositions:       &StaticPositions{},
		popularMap: &SortOverride{},

		idx: nil,
	}

	return instance

}

func (s *Sorting) GetSessionData(id uint) (*SortOverride, *SortOverride) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	itemData, ok := s.sessionOverrides[id]
	//fieldData, fieldOk := s.sessionFieldOverrides[id]
	if !ok {
		itemData = s.popularOverrides
	}
	//if !fieldOk {
	//	fieldData =
	//}
	return itemData, s.fieldOverride
}

func ListenForSessionMessage(rdb *redis.Client, channel string, fn func(sessionId int, sortOverride *SortOverride)) {
	ctx := context.Background()
	go func(sub *redis.PubSub) {
		for {
			msg, err := sub.ReceiveMessage(ctx)
			if err == nil {
				log.Printf("Received session message %s", msg.Payload)
				idx := strings.LastIndex(msg.Payload, "_")
				if idx == -1 {
					log.Println("Invalid session override change message", msg.Payload)
					continue
				}
				sessionIdString := msg.Payload[idx+1:]
				sessionId, err := strconv.Atoi(sessionIdString)
				if err != nil {
					log.Println(err)
					continue
				}
				if sessionId == 0 {
					continue
				}

				sortOverride, err := GetOverrideFromKey(rdb, msg.Payload)

				if err != nil {
					log.Println(err)
					continue
				}
				fn(sessionId, sortOverride)
			} else {
				log.Println("Error receiving session message", err)
			}

		}
	}(rdb.Subscribe(ctx, channel))
}

func GetOverrideFromKey(rdb *redis.Client, key string) (*SortOverride, error) {
	data, err := rdb.Get(context.Background(), key).Result()
	if err != nil {
		return nil, err
	}
	sortOverride := SortOverride{}
	err = sortOverride.FromString(data)
	return &sortOverride, err
}

func ListenForSortOverride(rdb *redis.Client, channel string, key string, fn func(sortOverride *SortOverride)) {
	ctx := context.Background()
	go func(sub *redis.PubSub) {
		for {
			_, err := sub.ReceiveMessage(ctx)
			if err == nil {

				sortOverride, err := GetOverrideFromKey(rdb, key)
				if err != nil {
					fmt.Println(err)
					continue
				}
				fn(sortOverride)
			} else {
				fmt.Println("Error receiving sort override message", err)
			}
		}
	}(rdb.Subscribe(ctx, channel))
}

func (s *Sorting) SetSessionItemOverride(sessionId int, sortOverride *SortOverride) {
	s.muOverride.Lock()
	defer s.muOverride.Unlock()
	s.sessionOverrides[uint(sessionId)] = sortOverride
	s.hasItemChanges = true
}

//func (s *Sorting) SetSessionFieldOverride(sessionId int, sortOverride *SortOverride) {
//	s.muOverride.Lock()
//	defer s.muOverride.Unlock()
//	s.sessionFieldOverrides[uint(sessionId)] = sortOverride
//}

func (s *Sorting) StartListeningForChanges() {
	rdb := s.client
	//ctx := context.Background()

	go ListenForSessionMessage(rdb, REDIS_SESSION_POPULAR_CHANGE, s.SetSessionItemOverride)
	//go ListenForSessionMessage(rdb, REDIS_SESSION_FIELD_CHANGE, s.SetSessionFieldOverride)

	go ListenForSortOverride(rdb, REDIS_POPULAR_CHANGE, REDIS_POPULAR_KEY, s.addPopularOverride)
	go ListenForSortOverride(rdb, REDIS_FIELD_CHANGE, REDIS_FIELD_KEY, s.setFieldSortOverride)

	// go func(sub *redis.PubSub) {
	// 	for {
	// 		_, err := sub.ReceiveMessage(ctx)
	// 		if err != nil {

	// 			//fmt.Println("Received static positions change", msg.Channel, msg.Payload)
	// 			data, err := rdb.Get(ctx, REDIS_STATIC_KEY).Result()
	// 			if err != nil {
	// 				fmt.Println(err)
	// 				continue
	// 			}
	// 			staticPositions := StaticPositions{}
	// 			err = staticPositions.FromString(data)
	// 			if err != nil {
	// 				fmt.Println(err)
	// 				continue
	// 			}
	// 			s.setStaticPositions(staticPositions)
	// 		}
	// 	}

	// }(rdb.Subscribe(ctx, REDIS_STATIC_CHANGE))
}

func (s *Sorting) IndexChanged(idx *Index) {
	s.idx = idx
	s.hasItemChanges = true
}

// func (s *Sorting) GetStaticPositions() StaticPositions {
// 	s.muStaticPos.RLock()
// 	defer s.muStaticPos.RUnlock()
// 	return *s.staticPositions
// }

// func (s *Sorting) SetStaticPositions(positions StaticPositions) error {
// 	s.setStaticPositions(positions)
// 	ctx := context.Background()
// 	data := positions.ToString()
// 	s.client.Set(ctx, REDIS_STATIC_KEY, data, 0)
// 	_, err := s.client.Publish(ctx, REDIS_STATIC_CHANGE, "external").Result()
// 	return err
// }

// func (s *Sorting) setStaticPositions(positions StaticPositions) {
// 	s.muStaticPos.Lock()
// 	defer s.muStaticPos.Unlock()
// 	s.staticPositions = &positions
// }

func (s *Sorting) InitializeWithIndex(idx *Index) {
	// ctx := context.Background()
	s.idx = idx

	popularOverride, err := GetOverrideFromKey(s.client, REDIS_POPULAR_KEY)
	if err == nil {
		s.muOverride.Lock()
		s.popularOverrides = popularOverride
		s.muOverride.Unlock()
	}

	fieldOverride, err := GetOverrideFromKey(s.client, REDIS_FIELD_KEY)
	if err == nil {
		s.muOverride.Lock()
		s.fieldOverride = fieldOverride
		s.muOverride.Unlock()
	}

	// staticData, err := s.client.Get(ctx, REDIS_STATIC_KEY).Result()
	// if err == nil {
	// 	staticPositions := StaticPositions{}
	// 	err = staticPositions.FromString(staticData)
	// 	if err == nil {
	// 		s.setStaticPositions(staticPositions)
	// 	}
	// }
	s.makeFieldSort(idx, *s.fieldOverride)
	s.makeItemSortMaps()
	s.hasItemChanges = false
	log.Println("Sorting initialized")
	s.quit = make(chan struct{})
	ticker := time.NewTicker(60 * time.Second)
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

func (s *Sorting) makeFieldSort(idx *Index, overrides SortOverride) {
	idx.Lock()
	defer idx.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	fieldMap := make(SortOverride)
	//l := len(idx.Facets)
	//i := 0
	//
	//sortMap := make(types.ByValue, l)
	//var base *types.BaseField
	//for _, item := range idx.Facets {
	//	base = item.GetBaseField()
	//	if base.HideFacet {
	//		continue
	//	}
	//	sortMap[i] = getFieldLookupValue(*base, overrides[base.Id])
	//	i++
	//}
	//
	//sortMap = sortMap[:i]
	//sort.Sort(sort.Reverse(sortMap))
	//

	sortMap := types.ByValue(slices.SortedFunc(func(yield func(value types.Lookup) bool) {
		var base *types.BaseField
		for id, item := range idx.Facets {
			base = item.GetBaseField()
			if base.HideFacet {
				continue
			}
			v := base.Priority + overrides[base.Id]
			fieldMap[id] = v
			if !yield(types.Lookup{
				Id:    id,
				Value: v,
			}) {
				break
			}

		}
	}, types.LookUpReversed))
	s.fieldMap = &fieldMap
	s.FieldSort = &sortMap
}

func (s *Sorting) Close() error {
	return s.client.Close()
}

func (s *Sorting) addPopularOverride(sort *SortOverride) {
	s.muOverride.Lock()
	defer s.muOverride.Unlock()
	s.popularOverrides = sort
	s.hasItemChanges = true
}

func (s *Sorting) AddPopularOverride(sort *SortOverride) {
	data := sort.ToString()
	ctx := context.Background()
	s.client.Set(ctx, REDIS_POPULAR_KEY, data, 0)
	_, err := s.client.Publish(ctx, REDIS_POPULAR_CHANGE, "external").Result()
	if err != nil {
		s.addPopularOverride(sort)
	}
}

func (s *Sorting) GetPopularOverrides() *SortOverride {
	s.muOverride.RLock()
	defer s.muOverride.RUnlock()
	return s.popularOverrides
}

func (s *Sorting) GetSorting(id string, sortChan chan *types.ByValue) {
	sortChan <- s.GetSort(id)
}

func (s *Sorting) GetSort(id string) *types.ByValue {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if sortMethod, ok := s.sortMethods[id]; ok {
		return sortMethod
	}
	for _, method := range s.sortMethods {
		return method
	}
	return &types.ByValue{}
}

func (s *Sorting) GetSortedItemsIterator(sessionId int, precalculated *types.ByValue, items *types.ItemList, start int, sortedItemsChan chan<- iter.Seq[types.Item], overrides ...SortOverride) {
	if precalculated != nil {
		c := 0
		fn := func(yield func(types.Item) bool) {
			for _, v := range *precalculated {
				if _, ok := (*items)[v.Id]; !ok {
					continue
				}
				if c < start {
					c++
					continue
				}
				item, ok := s.idx.Items[v.Id]

				if !ok {
					continue
				}

				if !yield(item) {
					break
				}
			}

		}
		sortedItemsChan <- fn
		return
	} else {
		ch := make(chan []types.Lookup)

		if sessionId > 0 {
			if sessionOverride, ok := s.sessionOverrides[uint(sessionId)]; ok {
				overrides = append(overrides, *sessionOverride)
			}
		}
		go makeSortForItems(*s.popularMap, items, ch, overrides...)
		c := 0
		fn := func(yield func(types.Item) bool) {
			defer close(ch)
			for _, v := range <-ch {
				if c < start {
					c++
					continue
				}
				item, ok := s.idx.Items[v.Id]
				if !ok {
					continue
				}
				if !yield(item) {
					break
				}
			}
		}
		sortedItemsChan <- fn
	}
}

func (s *Sorting) GetSortedFields(items []*JsonFacet) []*JsonFacet {

	//var sessionOverride *SortOverride
	//if sessionId > 0 {
	//	if o, ok := s.sessionOverrides[uint(sessionId)]; ok {
	//		sessionOverride = o
	//	}
	//}
	base := s.fieldMap
	slices.SortFunc(items, func(a, b *JsonFacet) int {
		return cmp.Compare(SumOverrides(b.Id, base), SumOverrides(a.Id, base))
	})
	return items
}

func ToSortedMap[K comparable](i map[K]float64) []K {
	return slices.SortedFunc(func(yield func(K) bool) {
		for key, _ := range i {
			if !yield(key) {
				break
			}
		}
	}, func(k K, k2 K) int {
		return cmp.Compare(i[k2], i[k])
	})
}

func SumOverrides(id uint, overrides ...*SortOverride) float64 {
	sum := 0.0
	for _, o := range overrides {
		if o != nil {
			v, ok := (*o)[id]
			if ok {
				sum += v
			}
		}
	}
	return sum
}

func makeSortForItems(m SortOverride, items *types.ItemList, ch chan []types.Lookup, overrides ...SortOverride) {

	ch <- slices.SortedFunc(func(yield func(types.Lookup) bool) {
		var value float64
		var ok bool
		var add float64
		for id := range *items {
			value, ok = m[id]
			if !ok {
				value = 0
			}
			add = 0
			for _, override := range overrides {
				if v, ok := override[id]; ok {
					add += v
				}
			}
			if !yield(types.Lookup{Id: id, Value: value + add}) {
				break
			}
		}
	}, types.LookUpReversed)
}

func (s *Sorting) makeItemSortMaps() {
	s.muOverride.RLock()
	defer s.muOverride.RUnlock()

	overrides := *s.popularOverrides

	s.idx.Lock()
	defer s.idx.Unlock()
	l := len(s.idx.Items)
	j := 0.0
	now := time.Now()
	ts := now.UnixMilli()
	popularMap := make(types.ByValue, l)
	priceMap := make(types.ByValue, l)
	updatedMap := make(types.ByValue, l)
	createdMap := make(types.ByValue, l)
	popularSearchMap := make(SortOverride)
	i := 0
	var item types.Item

	var id uint
	var popular float64
	var partPopular float64

	for id, item = range s.idx.Items {

		if item.IsDeleted() {
			continue
		}
		j += 0.0000000000001

		popular = item.GetBasePopularity() + (overrides[id] * 100)

		partPopular = popular / 10000.0
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
		popularSearchMap[id] = popular / 100.0
		i++
	}

	// if s.idx != nil {
	// 	s.idx.SetBaseSortMap(popularSearchMap)
	// }
	go func() {
		popularMap = popularMap[:i]
		priceMap = priceMap[:i]
		updatedMap = updatedMap[:i]
		createdMap = createdMap[:i]
		s.muOverride.Lock()
		defer s.muOverride.Unlock()
		s.mu.Lock()
		defer s.mu.Unlock()
		s.popularMap = &popularSearchMap
		SortByValues(popularMap)
		s.sortMethods[POPULAR_SORT] = &popularMap
		SortByValues(priceMap)
		s.sortMethods[PRICE_DESC_SORT] = &priceMap
		s.sortMethods[PRICE_SORT] = cloneReversed(&priceMap)
		SortByValues(updatedMap)
		slices.Reverse(updatedMap)
		//s.sortMethods[UPDATED_DESC_SORT] = &updatedMap
		s.sortMethods[UPDATED_SORT] = &updatedMap
		SortByValues(createdMap)
		s.sortMethods[CREATED_SORT] = &createdMap
		//s.sortMethods[CREATED_DESC_SORT] = cloneReversed(&createdMap)
	}()

}

func SortByValues(arr types.ByValue) {
	slices.SortFunc(arr, func(a, b types.Lookup) int {
		return cmp.Compare(b.Value, a.Value)
	})
}

func cloneReversed(arr *types.ByValue) *types.ByValue {
	n := make(types.ByValue, len(*arr))
	copy(n, *arr)
	slices.Reverse(n)
	return &n
}

func (s *Sorting) setFieldSortOverride(sort *SortOverride) {
	if s.idx != nil {
		go s.makeFieldSort(s.idx, *sort)
	}
	s.muOverride.Lock()
	defer s.muOverride.Unlock()
	s.fieldOverride = sort
}

// func (s *Sorting) SetFieldSortOverride(sort *SortOverride) {
// 	ctx := context.Background()
// 	data := sort.ToString()
// 	log.Printf("Setting field sort %d", len(data))
// 	s.client.Set(ctx, REDIS_FIELD_KEY, data, 0)
// 	err := s.client.Publish(ctx, REDIS_FIELD_CHANGE, REDIS_FIELD_KEY)
// 	if err != nil {
// 		s.setFieldSortOverride(sort)
// 	}

// }
