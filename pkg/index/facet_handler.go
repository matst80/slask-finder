package index

import (
	"log"
	"sync"

	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/types"
)

type FacetItemHandler struct {
	mu            sync.RWMutex
	ChangeHandler FieldChangeHandler
	Facets        map[uint]types.Facet
	ItemFieldIds  map[uint]types.ItemList
}

type FacetItemHandlerOptions struct {
	// Add any facet-specific options here if needed
}

func NewFacetItemHandler(opts FacetItemHandlerOptions) *FacetItemHandler {
	return &FacetItemHandler{
		Facets:       make(map[uint]types.Facet),
		ItemFieldIds: make(map[uint]types.ItemList),
	}
}

// ItemHandler interface implementation
func (h *FacetItemHandler) HandleItem(item types.Item) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.HandleItemUnsafe(item)
}

func (h *FacetItemHandler) HandleItems(items []types.Item) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, item := range items {
		h.HandleItemUnsafe(item)
	}
}

func (h *FacetItemHandler) HandleItemUnsafe(item types.Item) {
	h.removeItemValues(item)
	if item.IsDeleted() {
		return
	}

	h.addItemValues(item)
}

func (h *FacetItemHandler) Lock() {
	h.mu.Lock()
}

func (h *FacetItemHandler) Unlock() {
	h.mu.Unlock()
}

// Facet management methods
func (h *FacetItemHandler) AddKeyField(field *types.BaseField) {
	h.Facets[field.Id] = facet.EmptyKeyValueField(field)
}

func (h *FacetItemHandler) AddDecimalField(field *types.BaseField) {
	h.Facets[field.Id] = facet.EmptyDecimalField(field)
}

func (h *FacetItemHandler) AddIntegerField(field *types.BaseField) {
	h.Facets[field.Id] = facet.EmptyIntegerField(field)
}

func (h *FacetItemHandler) GetKeyFacet(id uint) (*facet.KeyField, bool) {
	if f, ok := h.Facets[id]; ok {
		switch tf := f.(type) {
		case facet.KeyField:
			return &tf, true
		case *facet.KeyField:
			return tf, true
		}
	}
	return nil, false
}

// Item processing methods
func (h *FacetItemHandler) addItemValues(item types.Item) {
	itemId := item.GetId()

	b := &types.BaseField{}
	for id, fieldValue := range item.GetFields() {
		if f, ok := h.Facets[id]; ok {
			b = f.GetBaseField()
			if b.Searchable && f.AddValueLink(fieldValue, itemId) && h.ItemFieldIds != nil {
				if !b.HideFacet {
					if fids, ok := h.ItemFieldIds[itemId]; ok {
						fids.AddId(id)
					} else {
						log.Printf("No field for item id: %d, id: %d", itemId, id)
					}
				}
			}
		}
	}
}

func (h *FacetItemHandler) removeItemValues(item types.Item) {
	itemId := item.GetId()

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
