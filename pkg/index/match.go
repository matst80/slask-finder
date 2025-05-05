package index

import (
	"cmp"
	"fmt"
	"log"
	"slices"

	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/types"
)

type CleanKeyFacet struct {
	Facet   *facet.KeyField
	level   int
	Exclude bool
	Value   types.StringFilterValue `json:"value"`
}

func (i *Index) RemoveDuplicateCategoryFilters(stringFilters []types.StringFilter) []CleanKeyFacet {
	ret := make([]CleanKeyFacet, 0, len(stringFilters))

	maxLevel := 0
	for _, fld := range stringFilters {
		if fld.Value == nil {
			continue
		}
		if f, ok := i.Facets[fld.Id]; ok && f != nil {
			keyFacet, ok := f.(*facet.KeyField)
			if !ok {
				log.Printf("Key facet %d not found", fld.Id)
				continue
			}
			level := f.GetBaseField().CategoryLevel
			ret = append(ret, CleanKeyFacet{
				Facet:   keyFacet,
				Exclude: fld.Not,
				Value:   fld.Value,
				level:   level,
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

func (i *Index) MatchStringsSync(filter []types.StringFilter, res *types.ItemList) {
	i.mu.Lock()
	defer i.mu.Unlock()
	first := true
	for _, fld := range filter {
		if f, ok := i.Facets[fld.Id]; ok && f != nil {
			keyFacet, ok := f.(*facet.KeyField)
			if !ok {
				log.Printf("Key facet %s not found", f.GetBaseField().Name)
				continue
			}
			ids := keyFacet.Match(fld.Value)

			if ids == nil {
				log.Printf("No ids for key facet %s, value %v", f.GetBaseField().Name, fld.Value)
				return
			}
			log.Printf("key facet %s, value %v, result length %v", f.GetBaseField().Name, fld.Value, len(*ids))

			if first {
				first = false
				res.Merge(ids)
			} else {
				res.Intersect(*ids)
			}
		}
	}
}

func (i *Index) Match(search *types.Filters, initialIds *types.ItemList, idList chan<- *types.ItemList) {
	cnt := 0
	i.mu.Lock()
	defer i.mu.Unlock()
	results := make(chan types.FilterResult)
	log.Printf("Search %+v", search)

	parseKeys := func(value types.StringFilterValue, exclude bool, f *facet.KeyField) {
		results <- types.FilterResult{
			Ids:     f.MatchFilterValue(value),
			Exclude: exclude,
		}
	}
	parseRange := func(field types.RangeFilter, f types.Facet) {
		results <- types.FilterResult{
			Ids:     f.Match(field),
			Exclude: false,
		}
	}
	excludeQueries := make([]CleanKeyFacet, 0)
	for _, fld := range i.RemoveDuplicateCategoryFilters(search.StringFilter) {
		// log.Printf("key facet %s, value %v", fld.GetBaseField().Name, fld.Value)
		if fld.Exclude {
			excludeQueries = append(excludeQueries, fld)
			continue
		}
		cnt++

		go parseKeys(fld.Value, false, fld.Facet)

	}

	for _, fld := range search.RangeFilter {
		if f, ok := i.Facets[fld.Id]; ok && f != nil {
			cnt++
			// log.Printf("range facet %s, value %v", f.GetBaseField().Name, fld)
			go parseRange(fld, f)
		}
	}
	for _, fld := range excludeQueries {
		cnt++
		go parseKeys(fld.Value, true, fld.Facet)
	}
	if initialIds != nil {
		if cnt == 0 {
			idList <- initialIds
			return
		}
		cnt++
		go func() {
			results <- types.FilterResult{
				Ids:     initialIds,
				Exclude: false,
			}
		}()
	}

	idList <- types.MakeIntersectResult(results, cnt)

}

type KeyFieldWithValue struct {
	*facet.KeyField
	Value types.StringFilterValue
}

func (i *Index) Compatible(id uint) (*types.ItemList, error) {
	i.Lock()
	item, ok := i.Items[id]
	i.Unlock()
	if !ok {
		return nil, fmt.Errorf("item with id %d not found", id)
	}
	fields := make([]KeyFieldWithValue, 0)
	result := types.ItemList{}
	var base *types.BaseField

	//types.CurrentSettings.RLock()
	rel := types.CurrentSettings.FacetRelations
	//types.CurrentSettings.RUnlock()
	hasRealRelations := false
	for _, relation := range rel {
		if relation.Matches(item) {
			// match all items for this relation

			relationResult := &types.ItemList{}

			i.MatchStringsSync(relation.GetFilter(item), relationResult)

			hasRealRelations = true
			log.Printf("Found relation %s for item %d", relation.Name, item.GetId())
			result.Merge(relationResult)

		} else {
			log.Printf("No relation %+v for item %+v", relation, item)
		}
	}
	if !hasRealRelations {
		log.Printf("No relations found for item %d", item.GetId())
		for id, itemField := range item.GetFields() {
			field, ok := i.Facets[id]

			if !ok || field.GetType() != types.FacetKeyType {
				continue
			}
			keyFacet, ok := field.(*facet.KeyField)
			if !ok {
				log.Printf("Key facet %d not found", id)
				continue
			}
			base = keyFacet.BaseField
			if base.LinkedId == 0 {
				continue
			}
			targetField, ok := i.Facets[base.LinkedId]
			if !ok {
				log.Printf("Target facet %d not found", base.LinkedId)
				continue
			}

			keyValue, ok := types.AsKeyFilterValue(itemField)
			if !ok {
				log.Printf("Not a valid key filter", id)
				continue
			}

			targetKeyFacet, ok := targetField.(*facet.KeyField)
			if !ok {
				log.Printf("Key facet %d not found", base.LinkedId)
				continue
			}

			if ok {
				fields = append(fields, KeyFieldWithValue{
					KeyField: targetKeyFacet,
					Value:    keyValue,
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
		keyFacet, ok := field.(*facet.KeyField)
		if !ok {
			log.Printf("Key facet %d not found", id)
			continue
		}
		keyValue, ok := types.AsKeyFilterValue(itemField)
		if !ok {
			log.Printf("Not a valid key filter %v", itemField)
			continue
		}
		base = field.GetBaseField()
		if (base.CategoryLevel > 0 && base.CategoryLevel != 1) || base.Type != "" || base.LinkedId != 0 {
			fields = append(fields, KeyFieldWithValue{
				KeyField: keyFacet,
				Value:    keyValue,
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
