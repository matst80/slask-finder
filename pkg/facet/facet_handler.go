package facet

import (
	"cmp"
	"encoding/json"
	"iter"
	"log"
	"maps"
	"slices"
	"sync"

	"github.com/matst80/slask-finder/pkg/messaging"
	"github.com/matst80/slask-finder/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

type FacetItemHandler struct {
	mu           sync.RWMutex
	sortMap      map[uint]float64
	sortValues   types.ByValue
	override     map[uint]float64
	Facets       map[uint]types.Facet
	ItemFieldIds map[uint]types.ItemList
	AllFacets    types.ItemList
}

func (h *FacetItemHandler) HandleFieldChanges(items []types.FieldChange) {
	h.UpdateFields(items)
}

func (h *FacetItemHandler) Connect(conn *amqp.Connection) {

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	err = messaging.ListenToTopic(ch, "global", "field_sort_override", func(d amqp.Delivery) error {
		var item types.SortOverrideUpdate
		if err := json.Unmarshal(d.Body, &item); err == nil {
			log.Printf("Got sort override")
			if item.Key == "popular-fields" {
				h.mu.Lock()
				h.override = item.Data
				log.Printf("Got field overrides")
				h.mu.Unlock()
			} else {
				log.Printf("Discarding field sort override %s", item.Key)
			}

		} else {
			log.Printf("Failed to unmarshal facet change message %v", err)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to listen to facet_change topic: %v", err)
	}
}

func NewFacetItemHandler(facets []StorageFacet) *FacetItemHandler {
	r := &FacetItemHandler{
		Facets:       make(map[uint]types.Facet),
		ItemFieldIds: make(map[uint]types.ItemList),
		sortMap:      make(map[uint]float64),
		mu:           sync.RWMutex{},
		override:     make(map[uint]float64),
		AllFacets:    make(types.ItemList),
		//All:          types.ItemList{},
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
			continue
		}
		if f.BaseField.Searchable || !f.BaseField.HideFacet {
			r.AllFacets.AddId(f.Id)
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
			v := base.Priority + j // + r.overrides[base.Id]
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

	return r
}

func (h *FacetItemHandler) GetFacet(id uint) (types.Facet, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	f, ok := h.Facets[id]
	return f, ok
}

func (h *FacetItemHandler) GetAll() iter.Seq[types.Facet] {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return func(yield func(types.Facet) bool) {
		for _, f := range h.Facets {
			if !yield(f) {
				return
			}
		}
	}
}

func (h *FacetItemHandler) HandleItem(item types.Item, wg *sync.WaitGroup) {
	wg.Go(func() {

		itemId := item.GetId()
		h.mu.Lock()
		defer h.mu.Unlock()
		h.ItemFieldIds[itemId] = types.ItemList{}
		if item.IsDeleted() {
			delete(h.ItemFieldIds, itemId)

			for fieldId, fieldValue := range item.GetStringFields() {
				if f, ok := h.Facets[fieldId]; ok {
					f.RemoveValueLink(fieldValue, itemId)
				}
			}
			for fieldId, fieldValue := range item.GetNumberFields() {
				if f, ok := h.Facets[fieldId]; ok {
					f.RemoveValueLink(fieldValue, itemId)
				}
			}

		} else {

			for fieldId, fieldValue := range item.GetStringFields() {
				if f, ok := h.Facets[fieldId]; ok {
					b := f.GetBaseField()
					if b.Searchable && f.AddValueLink(fieldValue, fieldId) {
						if !b.HideFacet {
							if fids, ok := h.ItemFieldIds[itemId]; ok {
								fids.AddId(fieldId)
							} else {
								log.Printf("No string field for item id: %d, fieldId: %d", itemId, fieldId)
							}
						}
					}
				}
			}
			for fieldId, fieldValue := range item.GetNumberFields() {
				if f, ok := h.Facets[fieldId]; ok {
					b := f.GetBaseField()
					if b.Searchable && f.AddValueLink(fieldValue, fieldId) {
						if !b.HideFacet {
							if fids, ok := h.ItemFieldIds[itemId]; ok {
								fids.AddId(fieldId)
							} else {
								log.Printf("No number field for item id: %d, id: %d", itemId, fieldId)
							}
						}
					}
				}
			}
		}
	})

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

func (h *FacetItemHandler) UpdateFields(changes []types.FieldChange) {
	h.mu.Lock()
	defer h.mu.Unlock()
	log.Printf("Updating facet fields %d", len(changes))
	for _, change := range changes {
		if change.Action == types.ADD_FIELD {
			switch change.FieldType {
			case 1:
				h.AddKeyField(change.BaseField)
			case 3:
				h.AddIntegerField(change.BaseField)
			case 2:
				h.AddDecimalField(change.BaseField)
			default:
				log.Printf("Unknown field type %d", change.FieldType)
			}
		} else {
			if f, ok := h.Facets[change.Id]; ok {
				switch change.Action {
				case types.UPDATE_FIELD:
					f.UpdateBaseField(change.BaseField)
				case types.REMOVE_FIELD:
					delete(h.Facets, change.Id)
				}
			}
		}
	}
}

func getFacetResult(f types.Facet, baseIds *types.ItemList, c chan *JsonFacet, wg *sync.WaitGroup, selected any) {
	defer wg.Done()

	returnAll := len(*baseIds) == 0

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
			if returnAll {
				hasValues = true
				r[key] = count
				continue
			}
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
		if returnAll {
			c <- &JsonFacet{
				BaseField: baseField,
				Selected:  selected,
				Result: &IntegerFieldResult{
					Min: field.Min,
					Max: field.Max,
				},
			}
		} else {
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
	case DecimalField:
		if returnAll {
			c <- &JsonFacet{
				BaseField: baseField,
				Selected:  selected,
				Result: &DecimalFieldResult{
					Min: field.Min,
					Max: field.Max,
				},
			}
		} else {
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

	if len(fieldIds) == 0 {
		for id := range ws.sortValues.SortMap(ws.AllFacets) {
			if count > limit {
				break
			}

			if !sr.Filters.HasField(id) && !sr.IsIgnored(id) {

			}
		}

	}

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
		}
	}

}
