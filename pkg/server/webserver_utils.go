package server

import (
	"fmt"
	"net/http"
	"strconv"

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
	}
	w.Header().Set("Age", "0")
}

func publicHeaders(w http.ResponseWriter, r *http.Request, isJson bool, cacheTime string) {
	w.Header().Set("Cache-Control", "public, max-age="+cacheTime)
	genericHeaders(w, r, isJson)
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

func (ws *WebServer) getCategoryItemIds(categories []string, sr *SearchRequest, categoryStartId uint) *types.ItemList {

	ch := make(chan *types.ItemList)
	sortChan := make(chan *types.SortIndex)
	defer close(sortChan)
	defer close(ch)
	for i := 0; i < len(categories); i++ {
		sr.Filters.StringFilter = append(sr.Filters.StringFilter, facet.StringFilter{
			Id:    categoryStartId + uint(i),
			Value: categories[i],
		})
	}
	go ws.Index.Match(sr.Filters, nil, ch)
	return <-ch
}

func getCacheKey(sr *SearchRequest) string {
	fields := sr.Query
	for _, f := range sr.Filters.StringFilter {
		fields += strconv.Itoa(int(f.Id)) + "_" + fmt.Sprintf("%v", f.Value)
	}
	for _, f := range sr.Filters.RangeFilter {
		fields += strconv.Itoa(int(f.Id)) + "_" + fmt.Sprintf("%v_%v", f.Min, f.Max)
	}
	// for _, f := range sr.Filters.IntegerFilter {
	// 	fields += strconv.Itoa(int(f.Id)) + "_" + strconv.Itoa(f.Min) + "_" + strconv.Itoa(f.Max)
	// }
	return fmt.Sprintf("facets_%s_%s", sr.Query, fields)
}
