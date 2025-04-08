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

//func removeEmptyStrings(s []string) []string {
//	var r []string
//	for _, str := range s {
//		if str != "" {
//			r = append(r, str)
//		}
//	}
//	return r
//}

func (ws *WebServer) getCategoryItemIds(categories []string, sr *SearchRequest, categoryStartId uint) *types.ItemList {

	ch := make(chan *types.ItemList)
	sortChan := make(chan *types.SortIndex)
	defer close(sortChan)
	defer close(ch)
	for i := 0; i < len(categories); i++ {
		sr.Filters.StringFilter = append(sr.Filters.StringFilter, types.StringFilter{
			Id:    categoryStartId + uint(i),
			Value: categories[i],
		})
	}
	go ws.Index.Match(sr.Filters, nil, ch)
	return <-ch
}

//func getCacheKey(sr *SearchRequest) string {
//	fields := sr.Query
//	for _, f := range sr.Filters.StringFilter {
//		fields += strconv.Itoa(int(f.Id)) + "_" + fmt.Sprintf("%v", f.Value)
//	}
//	for _, f := range sr.Filters.RangeFilter {
//		fields += strconv.Itoa(int(f.Id)) + "_" + fmt.Sprintf("%v_%v", f.Min, f.Max)
//	}
//	// for _, f := range sr.Filters.IntegerFilter {
//	// 	fields += strconv.Itoa(int(f.Id)) + "_" + strconv.Itoa(f.Min) + "_" + strconv.Itoa(f.Max)
//	// }
//	return fmt.Sprintf("facets_%s_%s", sr.Query, fields)
//}

func (ws *WebServer) getMatchAndSort(sr *SearchRequest, result chan<- searchResult) {
	matchingChan := make(chan *types.ItemList)
	sortChan := make(chan *types.ByValue)
	go noSearches.Inc()

	defer close(matchingChan)
	defer close(sortChan)

	initialIds, documentResult := ws.getInitialIds(sr.FacetRequest)
	go ws.Index.Match(sr.Filters, initialIds, matchingChan)
	isPopular := sr.Sort == "popular" || sr.Sort == ""

	if isPopular && sr.Query != "*" {
		go func() {
			sortChan <- nil
		}()
	} else {
		go ws.Sorting.GetSorting(sr.Sort, sortChan)
	}

	if documentResult != nil {
		queryOverride := index.SortOverride(*documentResult)
		result <- searchResult{
			matching:      <-matchingChan,
			sortOverrides: []index.SortOverride{queryOverride},
			sort:          <-sortChan,
		}
		return
	}
	result <- searchResult{
		matching:      <-matchingChan,
		sort:          <-sortChan,
		sortOverrides: []index.SortOverride{},
	}
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
			ret.Result = &index.IntegerFieldResult{
				Count: uint(field.Count),
				Min:   field.Min,
				Max:   field.Max,
			}
		case facet.DecimalField:
			ret.Result = &index.DecimalFieldResult{
				Count: uint(field.Count),
				Min:   field.Min,
				Max:   field.Max,
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
		var ok bool
		for keyId, sourceIds := range field.Keys {
			count = 0
			for id := range sourceIds {
				if _, ok = matchIds[id]; ok {
					count++
				}
			}
			if count > 0 {
				hasValues = true
				r[string(keyId)] = count
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
		fieldResult := index.IntegerFieldResult{
			Count: 0,
			Min:   9999999999999999,
			Max:   -9999999999999999,
		}
		hasValues := false
		for id := range matchIds {
			if value := field.ValueForItemId(id); value != nil {
				fieldResult.Count++
				hasValues = true
				if *value < fieldResult.Min {
					fieldResult.Min = *value
				}
				if *value > fieldResult.Max {
					fieldResult.Max = *value
				}
			}
		}
		if !hasValues {
			c <- nil
			return
		}
		ret.Result = &fieldResult
	case facet.DecimalField:
		fieldResult := index.DecimalFieldResult{
			Count: 0,
			Min:   9999999999999999,
			Max:   -9999999999999999,
		}
		hasResults := false

		for id := range matchIds {
			if value := field.ValueForItemId(id); value != nil {
				fieldResult.Count++
				hasResults = true
				if *value < fieldResult.Min {
					fieldResult.Min = *value
				}
				if *value > fieldResult.Max {
					fieldResult.Max = *value
				}
			}
		}
		if !hasResults {
			c <- nil
			return
		}
		ret.Result = &fieldResult
	}
	c <- modifyResult(ret)
}

func (ws *WebServer) getSearchedFacets(baseIds *types.ItemList, filters *types.Filters, ch chan *index.JsonFacet, wg *sync.WaitGroup) {
	for _, s := range filters.StringFilter {
		if f, ok := ws.Index.Facets[s.Id]; ok {
			if !f.GetBaseField().HideFacet {
				wg.Add(1)
				go func(otherFilters *types.Filters) {
					matchIds := make(chan *types.ItemList)
					defer close(matchIds)

					go ws.Index.Match(otherFilters, baseIds, matchIds)

					go getFacetResult(f, <-matchIds, ch, wg, func(facet *index.JsonFacet) *index.JsonFacet {
						if facet != nil {
							facet.Selected = s.Value
						}
						return facet
					})
				}(filters.WithOut(s.Id))
			}
		}
	}
	for _, r := range filters.RangeFilter {
		if f, ok := ws.Index.Facets[r.Id]; ok {
			wg.Add(1)
			go func(otherFilters *types.Filters) {
				matchIds := make(chan *types.ItemList)
				defer close(matchIds)
				go ws.Index.Match(otherFilters, baseIds, matchIds)
				go getFacetResult(f, <-matchIds, ch, wg, func(facet *index.JsonFacet) *index.JsonFacet {
					if facet != nil {
						facet.Selected = r
					}
					return facet
				})
			}(filters.WithOut(r.Id))
		}
	}
}

func (ws *WebServer) getOtherFacets(baseIds *types.ItemList, filters *types.Filters, ch chan *index.JsonFacet, wg *sync.WaitGroup) {

	fieldIds := make(map[uint]struct{})

	if len(*baseIds) > 65535 {
		for id, f := range ws.Index.Facets {
			if !f.GetBaseField().HideFacet {
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
	hasCat := filters.HasCategoryFilter()
	for id := range ws.Sorting.FieldSort.SortMap(fieldIds) {
		if count > 40 {
			break
		}

		if !filters.HasField(id) {
			if f, ok := ws.Index.Facets[id]; ok {
				base = f.GetBaseField()
				if base == nil || base.HideFacet {
					continue
				}
				if base.CategoryLevel > 0 && hasCat {
					continue
				}

				wg.Add(1)
				go getFacetResult(f, baseIds, ch, wg, func(facet *index.JsonFacet) *index.JsonFacet {
					if facet != nil && !facet.Result.HasValues() && base.CategoryLevel == 0 {
						return nil
					}
					return facet
				})
				if base.Type != "fps" {
					count++
				}
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
