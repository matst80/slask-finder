package index

import (
	"cmp"
	"fmt"
	"log"
	"slices"

	"github.com/matst80/slask-finder/pkg/types"
)

type CleanKeyFacet struct {
	types.Facet
	level int
	Value interface{} `json:"value"`
}

func (i *Index) RemoveDuplicateCategoryFilters(stringFilters []types.StringFilter) []CleanKeyFacet {
	ret := make([]CleanKeyFacet, 0, len(stringFilters))

	maxLevel := 0
	for _, fld := range stringFilters {
		if fld.Value == nil {
			continue
		}
		if f, ok := i.Facets[fld.Id]; ok && f != nil {
			level := f.GetBaseField().CategoryLevel
			ret = append(ret, CleanKeyFacet{
				Facet: f,
				Value: fld.Value,
				level: level,
			})

			if level > maxLevel {
				maxLevel = level
			}

		}
	}

	return slices.DeleteFunc(ret, func(f CleanKeyFacet) bool {
		return f.level > 0 && f.level < maxLevel
	})
}

func (i *Index) Match(search *types.Filters, initialIds *types.ItemList, idList chan<- *types.ItemList) {
	cnt := 0
	i.mu.Lock()
	defer i.mu.Unlock()
	results := make(chan *types.ItemList)
	// log.Printf("Search %+v", search)

	parseKeys := func(value interface{}, facet types.Facet) {
		results <- facet.Match(value)
	}
	parseRange := func(field types.RangeFilter, facet types.Facet) {
		results <- facet.Match(field)
	}

	for _, fld := range i.RemoveDuplicateCategoryFilters(search.StringFilter) {
		// log.Printf("key facet %s, value %v", fld.GetBaseField().Name, fld.Value)
		cnt++
		go parseKeys(fld.Value, fld.Facet)

	}

	for _, fld := range search.RangeFilter {
		if f, ok := i.Facets[fld.Id]; ok && f != nil {
			cnt++
			// log.Printf("range facet %s, value %v", f.GetBaseField().Name, fld)
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

func (i *Index) Compatible(id uint) (*types.ItemList, error) {
	i.Lock()
	defer i.Unlock()
	item, ok := i.Items[id]
	if !ok {
		return nil, fmt.Errorf("item with id %d not found", id)
	}
	fields := make([]KeyFieldWithValue, 0)
	result := types.ItemList{}
	var base *types.BaseField

	types.CurrentSettings.RLock()
	defer types.CurrentSettings.RUnlock()
	rel := types.CurrentSettings.FacetRelations
	hasRealRelations := false
	for _, relation := range rel {
		if relation.Matches(item) {
			// match all items for this relation
			ids := make(chan *types.ItemList)
			defer close(ids)
			go i.Match(relation.GetFilter(item), nil, ids)
			result.Merge(<-ids)
			hasRealRelations = true
			log.Printf("Found relation %s for item %d", relation.Name, item.GetId())
		}
	}
	if !hasRealRelations {
		log.Printf("No relations found for item %d", item.GetId())
		for id, itemField := range item.GetFields() {
			field, ok := i.Facets[id]
			if !ok || field.GetType() != types.FacetKeyType {
				continue
			}
			base = field.GetBaseField()
			if base.LinkedId == 0 {
				continue
			}
			targetField, ok := i.Facets[base.LinkedId]

			if ok {
				fields = append(fields, KeyFieldWithValue{
					Facet: targetField,
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

		result = types.ItemList{}
		for _, field := range fields {
			next := field.Match(field.Value)
			if next != nil {
				result.Merge(next)
				continue
			}
			// if next != nil && result.HasIntersection(next) {
			// 	result.Intersect(*next)
			// }
		}
	}
	return &result, nil
}

func (i *Index) Related(id uint) (*types.ItemList, error) {
	i.Lock()
	defer i.Unlock()
	item, ok := i.Items[id]
	if !ok {
		return nil, fmt.Errorf("item with id %d not found", id)
	}
	fields := make([]KeyFieldWithValue, 0)
	result := types.ItemList{}
	var base *types.BaseField
	for id, itemField := range item.GetFields() {
		field, ok := i.Facets[id]
		if !ok || field.GetType() != types.FacetKeyType {
			continue
		}
		base = field.GetBaseField()
		if (base.CategoryLevel > 0 && base.CategoryLevel != 1) || base.Type != "" || base.LinkedId != 0 {
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

	result = types.ItemList{}
	for _, field := range fields {
		next := field.Match(field.Value)
		if len(result) == 0 && next != nil {
			result.Merge(next)
			continue
		}
		if next != nil && result.HasIntersection(next) {
			result.Intersect(*next)
		}
		if len(result) <= 50 {
			break
		}
	}
	return &result, nil
}
