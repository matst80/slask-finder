package facet

import (
	"cmp"
	"iter"
	"log"
	"maps"
	"slices"
	"sync"

	"github.com/matst80/slask-finder/pkg/common"
	"github.com/matst80/slask-finder/pkg/types"
)

type queueItem struct {
	id      uint
	deleted bool
	values  map[uint]interface{}
}

type FacetItemHandler struct {
	mu           sync.RWMutex
	queue        *common.QueueHandler[queueItem]
	sortMap      map[uint]float64
	sortValues   types.ByValue
	Facets       map[uint]types.Facet
	ItemFieldIds map[uint]types.ItemList
	All          types.ItemList
}

const DefaultStorageName = "facets.json"

func LoadFacetsFromStorage(storage types.StorageProvider) ([]StorageFacet, error) {
	facets := []StorageFacet{}
	err := storage.LoadJson(&facets, DefaultStorageName)
	if err != nil {
		return facets, err
	}
	return facets, nil
}

func SaveFacetsToStorage(storage types.StorageProvider, facets []StorageFacet) error {
	return storage.SaveJson(facets, DefaultStorageName)
}

func NewFacetItemHandler(facets []StorageFacet) *FacetItemHandler {
	r := &FacetItemHandler{
		Facets:       make(map[uint]types.Facet),
		ItemFieldIds: make(map[uint]types.ItemList),
		sortMap:      make(map[uint]float64),
		mu:           sync.RWMutex{},
		All:          types.ItemList{},
	}

	for _, f := range facets {
		switch f.Type {
		case 1:
			r.AddKeyField(f.BaseField)
		case 3:
			r.AddIntegerField(f.BaseField)
		case 2:
			r.AddDecimalField(f.BaseField)
		default:
			log.Printf("Unknown field type %d", f.Type)
		}
	}

	r.sortValues = types.ByValue(slices.SortedFunc(func(yield func(value types.Lookup) bool) {
		var base *types.BaseField
		j := 0.0
		for id, item := range r.Facets {
			base = item.GetBaseField()
			if base.HideFacet {
				continue
			}
			v := base.Priority + j //+ overrides[base.Id]
			j += 0.000000000001
			r.sortMap[id] = v
			if !yield(types.Lookup{
				Id:    id,
				Value: v,
			}) {
				break
			}
		}
	}, types.LookUpReversed))

	r.queue = common.NewQueueHandler(r.processItems, 100)
	return r
}

func (h *FacetItemHandler) processItems(items []queueItem) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, item := range items {
		h.ItemFieldIds[item.id] = types.ItemList{}
		if item.deleted {
			delete(h.ItemFieldIds, item.id)
			for fieldId, fieldValue := range item.values {
				if f, ok := h.Facets[fieldId]; ok {
					f.RemoveValueLink(fieldValue, item.id)
				}
			}
			delete(h.All, item.id)
		} else {
			h.All.AddId(item.id)
			for id, fieldValue := range item.values {
				if f, ok := h.Facets[id]; ok {
					b := f.GetBaseField()
					if b.Searchable && f.AddValueLink(fieldValue, item.id) {
						if !b.HideFacet {
							if fids, ok := h.ItemFieldIds[item.id]; ok {
								fids.AddId(id)
							} else {
								log.Printf("No field for item id: %d, id: %d", item.id, id)
							}
						}
					}
				}
			}
		}
	}
}

// ItemHandler interface implementation
func (h *FacetItemHandler) HandleItem(item types.Item) {
	h.queue.Add(queueItem{
		id:      item.GetId(),
		values:  item.GetFields(),
		deleted: item.IsDeleted(),
	})
}

func toQueueItem(items iter.Seq[types.Item]) iter.Seq[queueItem] {
	return func(yield func(queueItem) bool) {
		for item := range items {
			if !yield(queueItem{
				id:      item.GetId(),
				values:  item.GetFields(),
				deleted: item.IsDeleted(),
			}) {
				return
			}
		}
	}
}

func (h *FacetItemHandler) HandleItems(items iter.Seq[types.Item]) {
	h.queue.AddIter(toQueueItem(items))
}

// Facet management methods
func (h *FacetItemHandler) AddKeyField(field *types.BaseField) {
	h.Facets[field.Id] = EmptyKeyValueField(field)
}

func (h *FacetItemHandler) AddDecimalField(field *types.BaseField) {
	h.Facets[field.Id] = EmptyDecimalField(field)
}

func (h *FacetItemHandler) AddIntegerField(field *types.BaseField) {
	h.Facets[field.Id] = EmptyIntegerField(field)
}

func (h *FacetItemHandler) GetKeyFacet(id uint) (*KeyField, bool) {
	if f, ok := h.Facets[id]; ok {
		switch tf := f.(type) {
		case KeyField:
			return &tf, true
		case *KeyField:
			return tf, true
		}
	}
	return nil, false
}

// func (h *FacetItemHandler) UpdateFields(changes []types.FieldChange) {
// 	h.mu.Lock()
// 	defer h.mu.Unlock()
// 	log.Printf("Updating facet fields %d", len(changes))
// 	for _, change := range changes {
// 		if change.Action == types.ADD_FIELD {
// 			log.Println("not implemented add field")
// 		} else {
// 			if f, ok := h.Facets[change.Id]; ok {
// 				switch change.Action {
// 				case types.UPDATE_FIELD:
// 					f.UpdateBaseField(change.BaseField)
// 				case types.REMOVE_FIELD:
// 					delete(h.Facets, change.Id)
// 				}
// 			}
// 		}
// 	}
// }

func getFacetResult(f types.Facet, baseIds *types.ItemList, c chan *JsonFacet, wg *sync.WaitGroup, selected interface{}) {
	defer wg.Done()

	baseField := f.GetBaseField()
	if baseField.HideFacet {
		c <- nil
		return
	}

	switch field := f.(type) {
	case KeyField:
		hasValues := false
		r := make(map[string]int, len(field.Keys))
		count := 0
		//var ok bool
		for key, sourceIds := range field.Keys {
			count = sourceIds.IntersectionLen(*baseIds)

			if count > 0 {
				hasValues = true
				r[key] = count
			}
		}
		if !hasValues {
			c <- nil
			return
		}
		c <- &JsonFacet{
			BaseField: baseField,
			Selected:  selected,
			Result: &KeyFieldResult{
				Values: r,
			},
		}

	case IntegerField:

		r := field.GetExtents(baseIds)
		if r == nil {
			c <- nil
			return
		}
		c <- &JsonFacet{
			BaseField: baseField,
			Selected:  selected,
			Result:    r,
		}
	case DecimalField:
		r := field.GetExtents(baseIds)
		if r == nil {
			c <- nil
			return
		}
		c <- &JsonFacet{
			BaseField: baseField,
			Selected:  selected,
			Result:    r,
		}

	}
}

func (ws *FacetItemHandler) GetSearchedFacets(baseIds *types.ItemList, sr *types.FacetRequest, ch chan *JsonFacet, wg *sync.WaitGroup) {

	makeQm := func(list *types.ItemList) *types.QueryMerger {
		qm := types.NewQueryMerger(list)
		if baseIds != nil {
			qm.Add(func() *types.ItemList {
				return baseIds
			})
		}
		return qm
	}
	for _, s := range sr.StringFilter {
		if sr.IsIgnored(s.Id) {
			continue
		}
		var f types.Facet
		var faceExists bool

		f, faceExists = ws.Facets[s.Id]

		if faceExists && !f.IsExcludedFromFacets() {

			wg.Add(1)

			go func(otherFilters *types.Filters) {
				matchIds := &types.ItemList{}
				qm := makeQm(matchIds)
				ws.Match(otherFilters, qm)
				qm.Wait()

				getFacetResult(f, matchIds, ch, wg, s.Value)
			}(sr.WithOut(s.Id, f.IsCategory()))

		}
	}
	for _, r := range sr.RangeFilter {
		var f types.Facet
		var facetExists bool

		f, facetExists = ws.Facets[r.Id]

		if facetExists && !sr.IsIgnored(r.Id) {
			wg.Add(1)
			go func(otherFilters *types.Filters) {
				matchIds := &types.ItemList{}
				qm := makeQm(matchIds)
				ws.Match(otherFilters, qm)
				qm.Wait()
				getFacetResult(f, matchIds, ch, wg, r)
			}(sr.WithOut(r.Id, false))
		}
	}
}

func (ws *FacetItemHandler) GetSuggestFacets(baseIds *types.ItemList, sr *types.FacetRequest, ch chan *JsonFacet, wg *sync.WaitGroup) {
	for _, id := range types.CurrentSettings.SuggestFacets {
		var f types.Facet
		var facetExists bool

		f, facetExists = ws.Facets[id]

		if facetExists && !f.IsExcludedFromFacets() {
			wg.Add(1)
			go getFacetResult(f, baseIds, ch, wg, nil)
		}
	}
}

func (ws *FacetItemHandler) SortJsonFacets(facets []*JsonFacet) {
	slices.SortFunc(facets, func(a, b *JsonFacet) int {
		return cmp.Compare(ws.sortMap[b.Id], ws.sortMap[a.Id])
	})
}

func (ws *FacetItemHandler) GetOtherFacets(baseIds *types.ItemList, sr *types.FacetRequest, ch chan *JsonFacet, wg *sync.WaitGroup) {

	fieldIds := make(map[uint]struct{})
	limit := 30
	resultCount := len(*baseIds)
	t := 0
	for id := range *baseIds {
		var itemFieldIds types.ItemList
		var ok bool

		itemFieldIds, ok = ws.ItemFieldIds[id]

		if ok {
			maps.Copy(fieldIds, itemFieldIds)
			t++
		}
		if t > 2500 {
			break
		}
	}

	count := 0
	//var base *types.BaseField = nil
	if resultCount == 0 {

		mainCat := ws.Facets[10] // todo setting

		if mainCat != nil {
			//base = mainCat.GetBaseField()
			wg.Add(1)
			go getFacetResult(mainCat, &ws.All, ch, wg, nil)
		}
	} else {

		for id := range ws.sortValues.SortMap(fieldIds) {
			if count > limit {
				break
			}

			if !sr.Filters.HasField(id) && !sr.IsIgnored(id) {

				f, facetExists := ws.Facets[id]

				if facetExists && !f.IsExcludedFromFacets() {

					wg.Add(1)
					go getFacetResult(f, baseIds, ch, wg, nil)

					count++

				}
			} else {
				// log.Printf("Facet %d is in filters", id)
			}
		}
	}
}
