package server

import (
	"fmt"
	"net/http"
	"strconv"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
)

const Origin = "*"

func defaultHeaders(w http.ResponseWriter, isJson bool, cacheTime string) {
	if isJson {
		w.Header().Set("Content-Type", "application/json")
	} else {
		w.Header().Set("Content-Type", "text/plain")
	}
	w.Header().Set("Cache-Control", "private, stale-while-revalidate="+cacheTime)
	w.Header().Set("Access-Control-Allow-Origin", Origin)
	w.Header().Set("Age", "0")
}

func publicHeaders(w http.ResponseWriter, isJson bool, cacheTime string) {
	if isJson {
		w.Header().Set("Content-Type", "application/json")
	} else {
		w.Header().Set("Content-Type", "text/plain")
	}
	w.Header().Set("Cache-Control", "public, max-age="+cacheTime)
	w.Header().Set("Access-Control-Allow-Origin", Origin)
	w.Header().Set("Age", "0")
}

func removeEmptyStrings(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func (ws *WebServer) getCategoryItemIds(categories []string, sr *SearchRequest, categoryStartId uint) *facet.IdList {

	ch := make(chan *facet.IdList)
	sortChan := make(chan *facet.SortIndex)
	defer close(sortChan)
	defer close(ch)
	for i := 0; i < len(categories); i++ {
		sr.Filters.StringFilter = append(sr.Filters.StringFilter, index.StringSearch{
			Id:    categoryStartId + uint(i),
			Value: categories[i],
		})
	}
	go ws.Index.Match(sr.Filters, nil, ch)
	return <-ch
}

func getFacetsForIds(matching *facet.IdList, index *index.Index, filters *index.Filters, fieldSort *facet.SortIndex, facetChan chan<- index.Facets) {
	facetChan <- index.GetFacetsFromResult(matching, filters, fieldSort)
}

func getCacheKey(sr *SearchRequest) string {
	fields := sr.Query
	for _, f := range sr.Filters.StringFilter {
		fields += strconv.Itoa(int(f.Id)) + "_" + f.Value
	}
	for _, f := range sr.Filters.NumberFilter {
		fields += strconv.Itoa(int(f.Id)) + "_" + strconv.FormatFloat(f.Min, 'f', -1, 64) + "_" + strconv.FormatFloat(f.Max, 'f', -1, 64)
	}
	for _, f := range sr.Filters.IntegerFilter {
		fields += strconv.Itoa(int(f.Id)) + "_" + strconv.Itoa(f.Min) + "_" + strconv.Itoa(f.Max)
	}
	return fmt.Sprintf("facets_%s_%s", sr.Query, fields)
}
