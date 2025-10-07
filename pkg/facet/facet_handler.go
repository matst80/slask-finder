package facet

import (
	"cmp"
	"encoding/json"
	"iter"
	"log"
	"slices"
	"sync"

	"github.com/RoaringBitmap/roaring/v2"
	"github.com/matst80/slask-finder/pkg/messaging"
	"github.com/matst80/slask-finder/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

type FacetItemHandler struct {
	mu           sync.RWMutex
	sortMap      types.SortOverride
	sortValues   types.ByValue
	override     types.SortOverride
	Facets       map[types.FacetId]types.Facet
	ItemFieldIds map[types.ItemId]*types.ItemList
	AllFacets    *types.ItemList
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
				h.mu.Unlock()
				log.Printf("Got field overrides")
				h.updateSortMap()
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

func NewFacetItemHandler(facets []types.StorageFacet, overrides *types.SortOverride) *FacetItemHandler {
	r := &FacetItemHandler{
		Facets:       make(map[types.FacetId]types.Facet),
		ItemFieldIds: make(map[types.ItemId]*types.ItemList),
		sortMap:      make(types.SortOverride),
		mu:           sync.RWMutex{},
		override:     make(types.SortOverride),
		AllFacets:    types.NewItemList(),
		//All:          types.ItemList{},
	}
	if overrides != nil {
		r.override = *overrides
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
			r.AllFacets.AddId(uint32(f.Id))
		}
	}

	r.updateSortMap()

	return r
}

func (h *FacetItemHandler) updateSortMap() {
	h.mu.Lock()
	defer h.mu.Unlock()
	values := types.ByValue(slices.SortedFunc(func(yield func(value types.Lookup) bool) {
		var base *types.BaseField
		j := 0.0
		for id, item := range h.Facets {
			base = item.GetBaseField()
			if base.HideFacet {
				continue
			}
			v := base.Priority + j + h.override[uint32(base.Id)]
			j += 0.000000000001
			h.sortMap[uint32(id)] = v
			if !yield(types.Lookup{
				Id:    uint32(id),
				Value: v,
			}) {
				break
			}
		}
	}, types.LookUpReversed))
	h.sortValues = values
}

func (h *FacetItemHandler) GetFacet(id types.FacetId) (types.Facet, bool) {
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
			fid, ok := h.ItemFieldIds[itemId]
			if !ok {
				fid = types.NewItemList()
				h.ItemFieldIds[itemId] = fid
			}

			for fieldId, fieldValue := range item.GetStringFields() {
				if f, ok := h.Facets[fieldId]; ok {
					b := f.GetBaseField()
					if b.Searchable && f.AddValueLink(fieldValue, itemId) {
						if !b.HideFacet {
							fid.AddId(uint32(fieldId))
						}
					}
				}
			}
			for fieldId, fieldValue := range item.GetNumberFields() {
				if f, ok := h.Facets[fieldId]; ok {
					b := f.GetBaseField()
					if b.Searchable && f.AddValueLink(fieldValue, itemId) {
						if !b.HideFacet {
							fid.AddId(uint32(fieldId))
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

func (h *FacetItemHandler) GetKeyFacet(id types.FacetId) (*KeyField, bool) {
	if f, ok := h.Facets[id]; ok {
		switch tf := f.(type) {
		case *KeyField:
			return tf, true

		case KeyField:
			return &tf, true

		default:
			log.Printf("not a key facet type %T", f)
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

	l := baseIds.Cardinality()
	returnAll := l == 0

	baseField := f.GetBaseField()
	if baseField.HideFacet {
		log.Printf("this should never been called field %d", baseField.Id)
		return
	}

	switch field := f.(type) {
	case *KeyField:
		hasValues := returnAll
		r := make(map[string]uint64, len(field.Keys))
		var count uint64
		for key, sourceIds := range field.Keys {
			if returnAll {
				r[key] = sourceIds.Cardinality()
			} else {
				count = sourceIds.IntersectionLen(baseIds)
				if count > 0 {
					hasValues = true
					r[key] = count
				}
			}
		}
		if !hasValues {
			return
		}
		c <- &JsonFacet{
			BaseField: baseField,
			Selected:  selected,
			Result: &KeyFieldResult{
				Values: r,
			},
		}

	case *IntegerField:
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
				return
			}
			c <- &JsonFacet{
				BaseField: baseField,
				Selected:  selected,
				Result:    r,
			}
		}
	case *DecimalField:

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
			c <- &JsonFacet{
				BaseField: baseField,
				Selected:  selected,
				Result:    r,
			}
		}
	default:
		log.Printf("unknown field type %T", field)
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
		return cmp.Compare(ws.sortMap[uint32(b.Id)], ws.sortMap[uint32(a.Id)])
	})
}

func (ws *FacetItemHandler) GetOtherFacets(baseIds *types.ItemList, sr *types.FacetRequest, ch chan *JsonFacet, wg *sync.WaitGroup) {

	fieldIds := roaring.Bitmap{}
	limit := 30

	baseIds.ForEach(func(id uint32) bool {
		itemFieldIds, ok := ws.ItemFieldIds[types.ItemId(id)]
		if !ok {
			return true
		}
		fieldIds.And(itemFieldIds.Bitmap())
		return fieldIds.GetCardinality() < 2500
	})

	count := 0

	if fieldIds.GetCardinality() == 0 {
		fieldIds = *ws.AllFacets.Bitmap()
	}

	for uid := range ws.sortValues.SortBitmap(fieldIds) {
		id := types.FacetId(uid)
		if count > limit {
			break
		}

		if !sr.Filters.HasField(id) && !sr.IsIgnored(id) {

			f, facetExists := ws.Facets[id]

			if facetExists && !f.IsExcludedFromFacets() {

				wg.Add(1)
				count++
				go getFacetResult(f, baseIds, ch, wg, nil)

			}
		}
	}

}
