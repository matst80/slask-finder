package server

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/tracking"
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
	defer close(ch)
	for i := 0; i < len(categories); i++ {
		sr.Filters.StringFilter = append(sr.Filters.StringFilter, index.StringSearch{
			Id:    categoryStartId + uint(i),
			Value: categories[i],
		})
	}

	go ws.matchQuery(sr, ch)
	return <-ch
}

func (ws *WebServer) matchQuery(sr *SearchRequest, ids chan<- *facet.IdList) {
	var initialIds *facet.IdList = nil
	if sr.Query != "" {
		queryResult := ws.Index.Search.Search(sr.Query)
		result := queryResult.ToResultWithSort()
		initialIds = result.IdList
	}

	ws.Index.Match(&sr.Filters, initialIds, ids)
}

func getSortedItems(matching *facet.IdList, index *index.Index, sort *facet.SortIndex, page int, pageSize int, itemChan chan<- []index.ResultItem) {
	ids := matching.SortedIds(sort, pageSize*(page+1))
	itemChan <- index.GetItems(ids, page, pageSize)
}

func getFacetsForIds(matching *facet.IdList, index *index.Index, filters *index.Filters, fieldSort *facet.SortIndex, facetChan chan<- index.Facets) {
	facetChan <- index.GetFacetsFromResult(matching, filters, fieldSort)
}

func getCacheKey(sr SearchRequest) string {
	fields := ""
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

func generateSessionId() int {
	return int(time.Now().UnixNano())
}

func setSessionCookie(w http.ResponseWriter, session_id int) {
	http.SetCookie(w, &http.Cookie{
		Name:  "sid",
		Value: fmt.Sprintf("%d", session_id),
		Path:  "/", //MaxAge: 7200
	})
}

func handleSessionCookie(tracking *tracking.ClickHouse, w http.ResponseWriter, r *http.Request) int {
	session_id := generateSessionId()
	c, err := r.Cookie("sid")
	if err != nil {
		// fmt.Printf("Failed to get cookie %v", err)
		if tracking != nil {
			go tracking.TrackSession(uint32(session_id), r)
		}
		setSessionCookie(w, session_id)

	} else {
		session_id, err = strconv.Atoi(c.Value)
		if err != nil {
			setSessionCookie(w, session_id)
		}
	}
	return session_id
}
