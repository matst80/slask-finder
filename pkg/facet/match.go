package facet

import (
	"context"
	"log"
	"slices"
	"strings"

	"github.com/matst80/slask-finder/pkg/types"
	"go.opentelemetry.io/otel"
)

type CleanKeyFacet struct {
	Facet   *KeyField
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

var (
	name   = "slask-finder-facets"
	tracer = otel.Tracer(name)
)

func SpannedFetcher(fn func() *types.ItemList, name string) func(ctx context.Context) *types.ItemList {
	return func(ctx context.Context) *types.ItemList {
		// Here you could add tracing or logging using the name parameter.
		_, span := tracer.Start(ctx, name)
		defer span.End()
		return fn()
	}
}

func (i *FacetItemHandler) MatchStringsSync(filter []types.StringFilter, qm *types.QueryMerger) {

	for _, fld := range filter {
		if keyFacet, ok := i.GetKeyFacet(fld.Id); ok {
			qm.Add(SpannedFetcher(func() *types.ItemList {
				return keyFacet.MatchFilterValue(fld.Value)
			}, "MatchStringsSync"))
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
			qm.Add(SpannedFetcher(func() *types.ItemList {
				return fld.Facet.MatchFilterValue(fld.Value)
			}, "Match filter value"))
		}
	}

	for _, fld := range search.RangeFilter {
		if f, ok := i.Facets[fld.Id]; ok && f != nil {
			qm.Add(SpannedFetcher(func() *types.ItemList {
				return f.Match(fld)
			}, "Match range filter"))
		}
	}

}

type KeyFieldWithValue struct {
	*KeyField
	Value types.StringFilterValue
}

func (i *FacetItemHandler) Compatible(ctx context.Context, item types.Item) (*types.ItemList, error) {

	//fields := make([]KeyFieldWithValue, 0)
	result := types.ItemList{}

	//types.CurrentSettings.RLock()
	rel := types.CurrentSettings.FacetRelations
	//types.CurrentSettings.RUnlock()

	outerMerger := types.NewCustomMerger(ctx, &result, func(ctx context.Context, current *types.ItemList, next *types.ItemList, isFirst bool) {
		current.Merge(next)
	})
	hasRealRelations := false
	for _, relation := range rel {
		if relation.Matches(item) {
			// match all items for this relation

			outerMerger.Add(func(ctx context.Context) *types.ItemList {
				_, span := tracer.Start(ctx, "FacetRelationMatch")
				defer span.End()
				relationResult := types.NewItemList()
				merger := types.NewQueryMerger(ctx, relationResult)
				i.MatchStringsSync(relation.GetFilter(item), merger)
				merger.Wait()
				return relationResult
			})

			if len(relation.Include) > 0 {
				outerMerger.Add(func(ctx context.Context) *types.ItemList {
					_, span := tracer.Start(ctx, "Includes")
					defer span.End()
					ret := types.NewItemList()
					for _, id := range relation.Include {
						ret.AddId(uint32(id))
					}
					return ret
				})
			}

			if len(relation.Exclude) > 0 {
				outerMerger.Exclude(func() *types.ItemList {
					ret := types.NewItemList()
					for _, id := range relation.Exclude {
						ret.AddId(uint32(id))
					}
					return ret
				})
			}

			hasRealRelations = true

		}
	}
	if hasRealRelations {
		outerMerger.Wait()
		if !result.IsEmpty() {
			return &result, nil
		}
	}
	mergedProperties := 0
	maybeMerger := types.NewCustomMerger(ctx, &result, func(ctx context.Context, current *types.ItemList, next *types.ItemList, isFirst bool) {
		if current.IsEmpty() && next != nil {
			current.Merge(next)
			mergedProperties++
			return
		}
		if next != nil && !next.IsEmpty() {
			l := current.IntersectionLen(next)
			if l >= 2 {
				current.Intersect(next)
				mergedProperties++
			} else {
				current.Merge(next)
			}
		}
	})
	log.Printf("No relations found for item %d", item.GetId())
	var field *KeyField
	var target *KeyField
	var ok bool
	for id, fieldValue := range item.GetStringFields() {

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

		// keyValue, ok := types.AsKeyFilterValue(itemField)
		// if !ok {
		// 	log.Printf("Not a valid key filter for field %d", id)
		// 	continue
		// }

		maybeMerger.Add(func(ctx context.Context) *types.ItemList {
			return target.MatchFilterValue(strings.Split(fieldValue, ";"))
		})

	}

	maybeMerger.Wait()
	if mergedProperties > 1 {
		return &result, nil
	}
	return &types.ItemList{}, nil
}

func (i *FacetItemHandler) Related(ctx context.Context, item types.Item) (*types.ItemList, error) {

	result := types.ItemList{}
	var base *types.BaseField
	qm := types.NewCustomMerger(ctx, &result, func(ctx context.Context, current *types.ItemList, next *types.ItemList, isFirst bool) {
		if current.Cardinality() < 10 && next != nil {
			result.Merge(next)
			return
		}
		if next != nil && result.IntersectionLen(next) > 10 {
			result.Intersect(next)
		}
	})
	for id, fieldValue := range item.GetStringFields() {
		field, ok := i.GetKeyFacet(id)
		if !ok {
			continue
		}

		base = field.GetBaseField()
		if (base.CategoryLevel > 0 && base.CategoryLevel != 1) || base.Type != "" || base.LinkedId != 0 {
			//if keyValue, ok := types.AsKeyFilterValue(itemField); ok {
			qm.Add(SpannedFetcher(func() *types.ItemList {
				return field.MatchFilterValue(strings.Split(fieldValue, ";"))
			}, "Related MatchFilterValue"))
			//}
		}
	}
	qm.Wait()
	return &result, nil
}
