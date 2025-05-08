package server

import (
	"encoding/json"
	"log"
	"maps"
	"net/http"
	"sync"

	"github.com/matst80/slask-finder/pkg/common"
	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/tracking"

	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/types"
)

func defaultHeaders(w http.ResponseWriter, r *http.Request, isJson bool, cacheTime string) {

	w.Header().Set("Cache-Control", "private, stale-while-revalidate="+cacheTime)
	genericHeaders(w, r, isJson)
}

func genericHeaders(w http.ResponseWriter, r *http.Request, isJson bool) {
	if isJson {
		w.Header().Set("Content-Type", "application/json")
	} else {
		w.Header().Set("Content-Type", "text/plain")
	}
	origin := r.Header.Get("Origin")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	w.Header().Set("Age", "0")
}

func publicHeaders(w http.ResponseWriter, r *http.Request, isJson bool, cacheTime string) {
	w.Header().Set("Cache-Control", "public, max-age="+cacheTime)
	genericHeaders(w, r, isJson)
}

func (ws *WebServer) getMatchAndSort(sr *SearchRequest, result chan<- searchResult) {
	ids := &types.ItemList{}
	sortChan := make(chan *types.ByValue)
	go noSearches.Inc()

	defer close(sortChan)
	qm := types.NewQueryMerger(ids)
	qm.Add(func() *types.ItemList {
		return ws.getSearchAndStockResult(sr.FacetRequest)
	})

	ws.Index.Match(sr.Filters, qm)
	isPopular := sr.Sort == "popular" || sr.Sort == ""

	if isPopular && sr.Query != "*" {
		go func() {
			sortChan <- nil
		}()
	} else {
		go ws.Sorting.GetSorting(sr.Sort, sortChan)
	}

	qm.Wait()
	result <- searchResult{
		matching:      ids,
		sort:          <-sortChan,
		sortOverrides: []index.SortOverride{},
	}
}

const (
	BucketSections = 20
)

type ResultBucket struct {
	bucket [BucketSections]uint
}

func (r *ResultBucket) AddValue(value uint) {
}

func getFacetResult(f types.Facet, baseIds *types.ItemList, c chan *index.JsonFacet, wg *sync.WaitGroup, modifyResult func(*index.JsonFacet) *index.JsonFacet) {
	defer wg.Done()
	if baseIds == nil || len(*baseIds) == 0 {
		baseField := f.GetBaseField()
		if baseField.HideFacet {
			c <- nil
			return
		}
		ret := &index.JsonFacet{
			BaseField: baseField,
		}
		switch field := f.(type) {
		case facet.KeyField:
			r := &index.KeyFieldResult{
				Values: make(map[string]uint),
			}
			for keyId, idList := range field.Keys {
				r.Values[string(keyId)] = uint(len(idList))
			}
			ret.Result = r
		case facet.IntegerField:
			ret.Result = &facet.IntegerFieldResult{
				//Count: uint(field.Max - field.Min),
				Min: field.Min,
				Max: field.Max,
			}
		case facet.DecimalField:
			ret.Result = &facet.DecimalFieldResult{
				//Count: uint(field.Count),
				Min: field.Min,
				Max: field.Max,
			}
		}
		c <- modifyResult(ret)
		return
	}
	matchIds := *baseIds
	baseField := f.GetBaseField()
	if baseField.HideFacet {
		c <- nil
		return
	}
	ret := &index.JsonFacet{
		BaseField: baseField,
	}
	switch field := f.(type) {
	case facet.KeyField:
		hasValues := false
		r := make(map[string]uint, len(field.Keys))
		count := uint(0)
		//var ok bool
		for key, sourceIds := range field.Keys {
			count = uint(sourceIds.IntersectionLen(matchIds))

			if count > 0 {
				hasValues = true
				r[key] = count
			}
		}
		if !hasValues {
			c <- nil
			return
		}
		ret.Result = &index.KeyFieldResult{
			Values: r,
		}
	case facet.IntegerField:

		r := field.GetExtents(matchIds)
		if r == nil {
			c <- nil
			return
		}
		ret.Result = r
	case facet.DecimalField:
		r := field.GetExtents(matchIds)
		if r == nil {
			c <- nil
			return
		}
		ret.Result = r

	}
	c <- modifyResult(ret)
}

func (ws *WebServer) getSearchedFacets(baseIds *types.ItemList, sr *types.FacetRequest, ch chan *index.JsonFacet, wg *sync.WaitGroup) {
	var base *types.BaseField
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
		if f, ok := ws.Index.Facets[s.Id]; ok {
			base = f.GetBaseField()

			if !base.HideFacet && !sr.IsIgnored(s.Id) {
				wg.Add(1)

				go func(otherFilters *types.Filters) {
					matchIds := &types.ItemList{}
					qm := makeQm(matchIds)
					ws.Index.Match(otherFilters, qm)
					qm.Wait()
					go getFacetResult(f, matchIds, ch, wg, func(facet *index.JsonFacet) *index.JsonFacet {
						if facet != nil {
							facet.Selected = s.Value
						}
						return facet
					})
				}(sr.WithOut(s.Id, base.CategoryLevel > 0))
			}
		}
	}
	for _, r := range sr.RangeFilter {
		if f, ok := ws.Index.Facets[r.Id]; ok && !sr.IsIgnored(r.Id) {
			wg.Add(1)
			go func(otherFilters *types.Filters) {
				matchIds := &types.ItemList{}
				qm := makeQm(matchIds)
				ws.Index.Match(otherFilters, qm)
				qm.Wait()
				go getFacetResult(f, matchIds, ch, wg, func(facet *index.JsonFacet) *index.JsonFacet {
					if facet != nil {
						facet.Selected = r
					}
					return facet
				})
			}(sr.WithOut(r.Id, false))
		}
	}
}

func (ws *WebServer) getSuggestFacets(baseIds *types.ItemList, sr *types.FacetRequest, ch chan *index.JsonFacet, wg *sync.WaitGroup) {

	for _, id := range types.CurrentSettings.SuggestFacets {
		if f, ok := ws.Index.Facets[id]; ok {
			base := f.GetBaseField()
			if base != nil {
				wg.Add(1)
				go getFacetResult(f, baseIds, ch, wg, func(facet *index.JsonFacet) *index.JsonFacet {
					// if facet != nil {
					// 	facet.Selected = s.Value
					// }
					return facet
				})
			}
		}
	}

}

func (ws *WebServer) getOtherFacets(baseIds *types.ItemList, sr *types.FacetRequest, ch chan *index.JsonFacet, wg *sync.WaitGroup) {

	fieldIds := make(map[uint]struct{})
	limit := 40
	resultCount := len(*baseIds)
	if resultCount > 65535 {
		limit = 20
		for id, f := range ws.Index.Facets {
			if !f.GetBaseField().HideFacet && !sr.IsIgnored(id) {
				fieldIds[id] = struct{}{}
			}
		}
	} else {
		for id := range *baseIds {
			itemFieldIds, ok := ws.Index.ItemFieldIds[id]
			if ok {
				maps.Copy(fieldIds, itemFieldIds)
			}
		}
	}
	count := 0
	var base *types.BaseField = nil
	if resultCount == 0 {
		mainCat := ws.Index.Facets[10] // todo setting
		if mainCat != nil {
			base = mainCat.GetBaseField()
			wg.Add(1)
			go getFacetResult(mainCat, &ws.Index.All, ch, wg, func(facet *index.JsonFacet) *index.JsonFacet {
				return facet
			})
		}
	} else {

		//hasCat := sr.Filters.HasCategoryFilter()
		for id := range ws.Sorting.FieldSort.SortMap(fieldIds) {
			if count > limit {
				break
			}

			if !sr.Filters.HasField(id) && !sr.IsIgnored(id) {
				if f, ok := ws.Index.Facets[id]; ok {
					base = f.GetBaseField()
					if base == nil || base.HideFacet {
						continue
					}
					// if base.CategoryLevel > 0 && hasCat {
					// 	continue
					// }

					wg.Add(1)
					go getFacetResult(f, baseIds, ch, wg, func(facet *index.JsonFacet) *index.JsonFacet {
						if facet != nil && !(facet.Result.HasValues() || facet.Type != "") && facet.CategoryLevel == 0 {
							return nil
						}
						return facet
					})
					if base.Type != "fps" {
						count++
					}
				}
			} else {
				// log.Printf("Facet %d is in filters", id)
			}
		}
	}
}

func JsonHandler(trk tracking.Tracking, fn func(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			RespondToOptions(w, r)
			return
		}
		sessionId := common.HandleSessionCookie(trk, w, r)

		err := fn(w, r, sessionId, json.NewEncoder(w))
		if err != nil {
			log.Printf("Error handling request: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
}

func RespondToOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=3600")
	origin := r.Header.Get("Origin")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	w.Header().Set("Age", "0")
	w.WriteHeader(http.StatusAccepted)
}
