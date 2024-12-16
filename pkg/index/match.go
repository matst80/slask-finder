package index

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/types"
)

type FilterIds map[uint]struct{}

type Filters struct {
	ids          *FilterIds
	StringFilter []facet.StringFilter `json:"string" schema:"-"`
	RangeFilter  []facet.RangeFilter  `json:"range" schema:"-"`
}

func (f *Filters) WithOut(id uint) *Filters {
	result := Filters{
		StringFilter: make([]facet.StringFilter, 0, len(f.StringFilter)),
		RangeFilter:  make([]facet.RangeFilter, 0, len(f.RangeFilter)),
	}
	for _, filter := range f.StringFilter {
		if filter.Id != id {
			result.StringFilter = append(result.StringFilter, filter)
		}
	}
	for _, filter := range f.RangeFilter {
		if filter.Id != id {
			result.RangeFilter = append(result.RangeFilter, filter)
		}
	}
	return &result
}

func (f *Filters) getIds() *FilterIds {
	if f.ids == nil {
		ids := make(FilterIds)
		if f.StringFilter != nil {
			for _, filter := range f.StringFilter {
				ids[filter.Id] = struct{}{}
			}
		}
		if f.RangeFilter != nil {
			for _, filter := range f.RangeFilter {
				ids[filter.Id] = struct{}{}
			}
		}
		f.ids = &ids
	}
	return f.ids
}

func (f *Filters) HasField(id uint) bool {
	ids := f.getIds()
	_, ok := (*ids)[id]
	return ok
}

func (f *Filters) HasCategoryFilter() bool {
	return slices.ContainsFunc(f.StringFilter, func(filter facet.StringFilter) bool {
		return filter.Id >= 30 && filter.Id <= 35 && filter.Id != 23
	})
}

func (i *Index) Match(search *Filters, initialIds *types.ItemList, idList chan<- *types.ItemList) {
	cnt := 0
	i.mu.Lock()
	defer i.mu.Unlock()
	results := make(chan *types.ItemList)

	parseKeys := func(field facet.StringFilter, fld types.Facet) {
		results <- fld.Match(field.Value)
	}
	parseRange := func(field facet.RangeFilter, fld types.Facet) {
		results <- fld.Match(field)
	}

	for _, fld := range search.StringFilter {
		if f, ok := i.Facets[fld.Id]; ok {
			cnt++
			go parseKeys(fld, f)
		}
	}

	for _, fld := range search.RangeFilter {
		if f, ok := i.Facets[fld.Id]; ok {
			cnt++
			go parseRange(fld, f)
		}
	}
	if initialIds != nil {
		if cnt == 0 {
			idList <- initialIds
			return
		}
		cnt++
		go func() {
			results <- initialIds
		}()
	}

	idList <- types.MakeIntersectResult(results, cnt)

}

type KeyFieldWithValue struct {
	types.Facet
	Value interface{}
}

func (i *Index) Related(id uint) (*types.ItemList, error) {
	i.Lock()
	defer i.Unlock()
	item, ok := i.Items[id]
	if !ok {
		return nil, fmt.Errorf("Item with id %d not found", id)
	}
	fields := make([]KeyFieldWithValue, 0)
	result := types.ItemList{}
	var base *types.BaseField
	for id, itemField := range (*item).GetFields() {
		field, ok := i.Facets[id]
		if !ok || field.GetType() != types.FacetKeyType {
			continue
		}
		base = field.GetBaseField()
		if base.CategoryLevel != 1 {
			fields = append(fields, KeyFieldWithValue{
				Facet: field,
				Value: itemField,
			})
		}
	}
	slices.SortFunc(fields, func(a, b KeyFieldWithValue) int {
		return cmp.Compare(b.GetBaseField().Priority, a.GetBaseField().Priority)
	})
	if len(fields) == 0 {
		return &result, nil
	}

	first := fields[0]
	result = *first.Match(first.Value)
	for _, field := range fields[1:] {
		if len(result) < 500 {
			return &result, nil
		}
		next := field.Match(field.Value)
		result.Intersect(*next)
	}
	return &result, nil
}
