package index

import (
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

func (i *Index) MatchStringsSync(filter []types.StringFilter, res *types.ItemList) {
	i.mu.Lock()
	defer i.mu.Unlock()
	qm := types.NewQueryMerger(res)

	for _, fld := range filter {
		if keyFacet, ok := i.GetKeyFacet(fld.Id); ok {
			qm.Add(func() *types.ItemList {
				return keyFacet.MatchFilterValue(fld.Value)
			})
		}
	}
	qm.Wait()
}

func (i *Index) Match(search *types.Filters, initialIds *types.ItemList, idList chan<- *types.ItemList) {

	log.Printf("Search %+v", search)
	result := make(types.ItemList)
	qm := types.NewQueryMerger(&result)
	if initialIds != nil {
		qm.Add(func() *types.ItemList {
			return initialIds
		})
	}

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

	qm.Wait()
	idList <- &result

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
	//fields := make([]KeyFieldWithValue, 0)
	result := types.ItemList{}

	//types.CurrentSettings.RLock()
	rel := types.CurrentSettings.FacetRelations
	//types.CurrentSettings.RUnlock()
	hasRealRelations := false

	outerMerger := types.NewCustomMerger(&result, func(current *types.ItemList, next *types.ItemList, isFirst bool) {
		current.Merge(next)
	})

	for _, relation := range rel {
		if relation.Matches(item) {
			// match all items for this relation

			outerMerger.Add(func() *types.ItemList {
				relationResult := &types.ItemList{}
				i.MatchStringsSync(relation.GetFilter(item), relationResult)
				return relationResult
			})

			if len(relation.Include) > 0 {
				outerMerger.Add(func() *types.ItemList {
					ret := &types.ItemList{}
					for _, id := range relation.Include {
						ret.AddId(id)
					}
					return ret
				})
			}

			if len(relation.Exclude) > 0 {
				outerMerger.Exclude(func() *types.ItemList {
					ret := &types.ItemList{}
					for _, id := range relation.Exclude {
						ret.AddId(id)
					}
					return ret
				})
			}

			hasRealRelations = true

		}
	}
	if !hasRealRelations {
		log.Printf("No relations found for item %d", item.GetId())
		for id, itemField := range item.GetFields() {

			field, ok := i.GetKeyFacet(id)

			if !ok {
				continue
			}

			linkedTo := field.BaseField.LinkedId
			if linkedTo == 0 {
				continue
			}
			targetField, ok := i.GetKeyFacet(linkedTo)
			if !ok {
				continue
			}

			keyValue, ok := types.AsKeyFilterValue(itemField)
			if !ok {
				log.Printf("Not a valid key filter", id)
				continue
			}

			outerMerger.Add(func() *types.ItemList {
				return targetField.MatchFilterValue(keyValue)
			})

		}

	}
	outerMerger.Wait()
	return &result, nil
}

func (i *Index) Related(id uint) (*types.ItemList, error) {
	i.Lock()
	item, ok := i.Items[id]
	i.Unlock()
	if !ok {
		return nil, fmt.Errorf("item with id %d not found", id)
	}

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
