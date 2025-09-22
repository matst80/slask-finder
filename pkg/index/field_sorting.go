package index

import (
	"slices"
	"sync"

	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/types"
	"github.com/redis/go-redis/v9"
)

// FieldSorting handles all field-related sorting functionality
type FieldSorting struct {
	mu            sync.RWMutex
	fieldOverride *SortOverride
	fieldMap      *SortOverride
	FieldSort     *types.ByValue
	client        *redis.Client
}

// NewFieldSorting creates a new FieldSorting instance
func NewFieldSorting(client *redis.Client) *FieldSorting {
	return &FieldSorting{
		fieldOverride: &SortOverride{},
		fieldMap:      &SortOverride{},
		FieldSort:     &types.ByValue{},
		client:        client,
	}
}

// makeFieldSort creates the field sorting based on facet data and overrides
func (fs *FieldSorting) makeFieldSort(idx *facet.FacetItemHandler, overrides SortOverride) {
	idx.Lock()
	defer idx.Unlock()
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	fieldMap := make(SortOverride)

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
	
	fs.fieldMap = &fieldMap
	fs.FieldSort = &sortMap
}

// setFieldSortOverride updates the field sort override
func (fs *FieldSorting) setFieldSortOverride(sort *SortOverride, facetIndex *facet.FacetItemHandler) {
	if facetIndex != nil {
		go fs.makeFieldSort(facetIndex, *sort)
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.fieldOverride = sort
}

// GetFieldOverride returns the current field override
func (fs *FieldSorting) GetFieldOverride() *SortOverride {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.fieldOverride
}

// GetFieldSort returns the current field sort
func (fs *FieldSorting) GetFieldSort() *types.ByValue {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.FieldSort
}

// InitializeFieldSort initializes the field sort with facet index and override
func (fs *FieldSorting) InitializeFieldSort(facetIndex *facet.FacetItemHandler, fieldOverride *SortOverride) {
	if fieldOverride != nil {
		fs.fieldOverride = fieldOverride
	}
	if facetIndex != nil {
		fs.makeFieldSort(facetIndex, *fs.fieldOverride)
	}
}