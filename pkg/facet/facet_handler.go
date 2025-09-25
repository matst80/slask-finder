package facet

import (
	"log"
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
	Facets       map[uint]types.Facet
	ItemFieldIds map[uint]types.ItemList
	queue        *common.QueueHandler[queueItem]
}

type FacetItemHandlerOptions struct {
	// Add any facet-specific options here if needed
}

func NewFacetItemHandler(opts FacetItemHandlerOptions) *FacetItemHandler {
	r := &FacetItemHandler{
		Facets:       make(map[uint]types.Facet),
		ItemFieldIds: make(map[uint]types.ItemList),
	}
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

		} else {
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

// // Item processing methods
// func (h *FacetItemHandler) addItemValues(item types.Item) {
// 	itemId := item.GetId()
// 	h.ItemFieldIds[itemId] = types.ItemList{}
// 	b := &types.BaseField{}
// 	for id, fieldValue := range item.GetFields() {
// 		if f, ok := h.Facets[id]; ok {
// 			b = f.GetBaseField()
// 			if b.Searchable && f.AddValueLink(fieldValue, itemId) {
// 				if !b.HideFacet {
// 					if fids, ok := h.ItemFieldIds[itemId]; ok {
// 						fids.AddId(id)
// 					} else {
// 						log.Printf("No field for item id: %d, id: %d", itemId, id)
// 					}
// 				}
// 			}
// 		}
// 	}
// }

func (h *FacetItemHandler) removeItemValues(item types.Item) {
	itemId := item.GetId()
	delete(h.ItemFieldIds, itemId)
	for fieldId, fieldValue := range item.GetFields() {
		if f, ok := h.Facets[fieldId]; ok {
			f.RemoveValueLink(fieldValue, itemId)
		}
	}
}

func (h *FacetItemHandler) UpdateFields(changes []types.FieldChange) {
	h.mu.Lock()
	defer h.mu.Unlock()
	log.Printf("Updating facet fields %d", len(changes))
	for _, change := range changes {
		if change.Action == types.ADD_FIELD {
			log.Println("not implemented add field")
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
