package index

import (
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

func (i *FacetItemHandler) RemoveDuplicateCategoryFilters(stringFilters []types.StringFilter) []CleanKeyFacet {
	ret := make([]CleanKeyFacet, 0, len(stringFilters))

	maxLevel := 0
	for _, fld := range stringFilters {
		if fld.Value == nil {
			continue
		}

		if f, ok := i.GetKeyFacet(fld.Id); ok {

			level := f.GetBaseField().CategoryLevel
			ret = append(ret, CleanKeyFacet{
				Facet:   f,
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

func (i *FacetItemHandler) MatchStringsSync(filter []types.StringFilter, qm *types.QueryMerger) {

	for _, fld := range filter {
		if keyFacet, ok := i.GetKeyFacet(fld.Id); ok {
			qm.Add(func() *types.ItemList {
				return keyFacet.MatchFilterValue(fld.Value)
			})
		}
	}

}

func (i *FacetItemHandler) Match(search *types.Filters, qm *types.QueryMerger) {

	for _, fld := range i.RemoveDuplicateCategoryFilters(search.StringFilter) {
		if fld.Exclude {
			qm.Exclude(func() *types.ItemList {

				return fld.Facet.MatchFilterValue(fld.Value)
			})
		} else {
			qm.Add(func() *types.ItemList {

				return fld.Facet.MatchFilterValue(fld.Value)
			})
		}
	}

	for _, fld := range search.RangeFilter {
		if f, ok := i.Facets[fld.Id]; ok && f != nil {
			qm.Add(func() *types.ItemList {
				return f.Match(fld)
			})
		}
	}

}

type KeyFieldWithValue struct {
	*facet.KeyField
	Value types.StringFilterValue
}

func (i *FacetItemHandler) Compatible(item types.Item) (*types.ItemList, error) {

	//fields := make([]KeyFieldWithValue, 0)
	result := types.ItemList{}

	//types.CurrentSettings.RLock()
	rel := types.CurrentSettings.FacetRelations
	//types.CurrentSettings.RUnlock()

	outerMerger := types.NewCustomMerger(&result, func(current *types.ItemList, next *types.ItemList, isFirst bool) {
		current.Merge(next)
	})
	hasRealRelations := false
	for _, relation := range rel {
		if relation.Matches(item) {
			// match all items for this relation

			outerMerger.Add(func() *types.ItemList {
				relationResult := make(types.ItemList, 5000)
				merger := types.NewQueryMerger(&relationResult)
				i.MatchStringsSync(relation.GetFilter(item), merger)
				merger.Wait()
				return &relationResult
			})

			if len(relation.Include) > 0 {
				outerMerger.Add(func() *types.ItemList {
					ret := make(types.ItemList, 5000)
					for _, id := range relation.Include {
						ret.AddId(id)
					}
					return &ret
				})
			}

			if len(relation.Exclude) > 0 {
				outerMerger.Exclude(func() *types.ItemList {
					ret := make(types.ItemList, 5000)
					for _, id := range relation.Exclude {
						ret.AddId(id)
					}
					return &ret
				})
			}

			hasRealRelations = true

		}
	}
	if hasRealRelations {
		outerMerger.Wait()
		if len(result) > 0 {
			return &result, nil
		}
	}
	mergedProperties := 0
	maybeMerger := types.NewCustomMerger(&result, func(current *types.ItemList, next *types.ItemList, isFirst bool) {
		if len(*current) == 0 && next != nil {
			current.Merge(next)
			mergedProperties++
			return
		}
		if next != nil && len(*next) > 0 {
			l := current.IntersectionLen(*next)
			if l >= 2 {
				current.Intersect(*next)
				mergedProperties++
			} else {
				current.Merge(next)
			}
		}
	})
	log.Printf("No relations found for item %d", item.GetId())
	var field *facet.KeyField
	var target *facet.KeyField
	var ok bool
	for id, itemField := range item.GetFields() {

		if field, ok = i.GetKeyFacet(id); !ok {
			continue
		}

		linkedTo := field.BaseField.LinkedId
		if linkedTo == 0 {
			continue
		}
		if target, ok = i.GetKeyFacet(linkedTo); !ok {
			continue
		}

		keyValue, ok := types.AsKeyFilterValue(itemField)
		if !ok {
			log.Printf("Not a valid key filter for field %d", id)
			continue
		}

		maybeMerger.Add(func() *types.ItemList {
			return target.MatchFilterValue(keyValue)
		})

	}

	maybeMerger.Wait()
	if mergedProperties > 1 {
		return &result, nil
	}
	return &types.ItemList{}, nil
}

func (i *FacetItemHandler) Related(item types.Item) (*types.ItemList, error) {

	result := types.ItemList{}
	var base *types.BaseField
	qm := types.NewCustomMerger(&result, func(current, next *types.ItemList, isFirst bool) {
		if len(*current) < 10 && next != nil {
			result.Merge(next)
			return
		}
		if next != nil && result.IntersectionLen(*next) > 10 {
			result.Intersect(*next)
		}
	})
	for id, itemField := range item.GetFields() {
		field, ok := i.GetKeyFacet(id)
		if !ok {
			continue
		}

		base = field.GetBaseField()
		if (base.CategoryLevel > 0 && base.CategoryLevel != 1) || base.Type != "" || base.LinkedId != 0 {
			if keyValue, ok := types.AsKeyFilterValue(itemField); ok {
				qm.Add(func() *types.ItemList {
					return field.MatchFilterValue(keyValue)
				})
			}
		}
	}
	qm.Wait()
	return &result, nil
}
