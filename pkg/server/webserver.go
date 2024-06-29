package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"strconv"
	"strings"
	"time"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/persistance"
	"tornberg.me/facet-search/pkg/tracking"
)

type WebServer struct {
	Index            *index.Index
	Db               *persistance.Persistance
	DefaultSort      *facet.SortIndex
	SortMethods      map[string]*facet.SortIndex
	FieldSort        *facet.SortIndex
	Cache            *Cache
	Tracking         *tracking.ClickHouse
	FacetLimit       int
	SearchFacetLimit int
	ListenAddress    string
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

func (ws *WebServer) Search(w http.ResponseWriter, r *http.Request) {
	session_id := handleSessionCookie(ws.Tracking, w, r)
	sr, err := QueryFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cacheHelper := NewCacheHelper[index.Facets](ws.Cache)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	itemsChan := make(chan []index.ResultItem)
	facetsChan := make(chan index.Facets)
	defer close(itemsChan)
	defer close(facetsChan)
	var initialIds *facet.IdList = nil
	itemSort := ws.DefaultSort
	if sr.Sort != "" {
		s, ok := ws.SortMethods[sr.Sort]
		if ok {
			itemSort = s
		}
	}
	if sr.Query != "" {
		queryResult := ws.Index.Search.Search(sr.Query)
		result := queryResult.ToResultWithSort()
		initialIds = result.IdList
		itemSort = &result.SortIndex
	}
	matching := ws.Index.Match(&sr.Filters, initialIds)

	totalHits := len(*matching)
	go func() {
		itemsChan <- ws.Index.GetItems(matching.SortedIds(itemSort, sr.PageSize*(sr.Page+1)), sr.Page, sr.PageSize)
	}()
	go func() {
		if totalHits > ws.FacetLimit {
			if ws.Cache == nil {
				facetsChan <- ws.Index.DefaultFacets
				return
			}
			var result = &index.Facets{}
			cacheKey := getCacheKey(sr)
			cacheHelper.Handle(cacheKey, result, func() index.Facets {
				return ws.Index.GetFacetsFromResult(matching, &sr.Filters, ws.FieldSort)
			}, time.Second*3600)
			// err := ws.Cache.Get(cacheKey, result)
			// if err != nil {
			// 	r := ws.Index.GetFacetsFromResult(matching, &sr.Filters, ws.FieldSort)
			// 	facetsChan <- r
			// 	ws.Cache.Set(cacheKey, r, time.Second*3600)
			// 	return
			// }
			facetsChan <- *result

		} else {
			facetsChan <- ws.Index.GetFacetsFromResult(matching, &sr.Filters, ws.FieldSort)
		}
	}()
	go func() {
		if ws.Tracking != nil {
			err := ws.Tracking.TrackSearch(uint32(session_id), &sr.Filters, sr.Query)
			if err != nil {
				fmt.Printf("Failed to track search %v", err)
			}
		}
	}()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, stale-while-revalidate=20")
	w.Header().Set("Access-Control-Allow-Origin", Origin)
	w.Header().Set("Age", "0")
	w.WriteHeader(http.StatusOK)

	data := SearchResponse{
		Items:     <-itemsChan,
		Facets:    <-facetsChan,
		Page:      sr.Page,
		PageSize:  sr.PageSize,
		TotalHits: totalHits,
	}

	encErr := json.NewEncoder(w).Encode(data)
	if encErr != nil {
		http.Error(w, encErr.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) AddItem(w http.ResponseWriter, r *http.Request) {
	items := AddItemRequest{}
	err := json.NewDecoder(r.Body).Decode(&items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, item := range items {
		ws.Index.UpsertItem(&item)
	}
	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) Save(w http.ResponseWriter, r *http.Request) {
	err := ws.Db.SaveIndex(ws.Index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

const Origin = "*"

type SuggestResult struct {
	Word string `json:"match"`
	Hits int    `json:"hits"`
}

func (ws *WebServer) Suggest(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	suggestions := ws.Index.AutoSuggest.FindMatches(query)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, stale-while-revalidate=120")
	w.Header().Set("Access-Control-Allow-Origin", Origin)
	w.Header().Set("Age", "0")
	w.WriteHeader(http.StatusOK)
	result := make([]SuggestResult, len(suggestions))
	for i, s := range suggestions {
		result[i] = SuggestResult{
			Word: s.Word,
			Hits: len(*s.Ids),
		}
	}
	err := json.NewEncoder(w).Encode(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) QueryIndex(w http.ResponseWriter, r *http.Request) {

	itemsChan := make(chan []index.ResultItem)
	facetsChan := make(chan index.Facets)
	defer close(itemsChan)
	defer close(facetsChan)
	qs := r.URL.Query()
	query := qs.Get("q")
	ps := qs.Get("size")
	pg := qs.Get("p")

	pageSize, sizeErr := strconv.Atoi(ps)
	if sizeErr != nil {
		pageSize = 50
	}
	page, pageError := strconv.Atoi(pg)
	if pageError != nil {
		page = 0
	}

	//	log.Printf("Query: %v", query)
	searchResults := ws.Index.Search.Search(query)
	res := searchResults.ToResultWithSort()
	go func() {

		ids := res.SortIndex.SortMap(*res.IdList, (page+1)*pageSize)
		itemsChan <- ws.Index.GetItems(ids, page, pageSize)
	}()
	go func() {

		if len(*searchResults) > ws.SearchFacetLimit {
			facetsChan <- index.Facets{}
		} else {
			facetsChan <- ws.Index.GetFacetsFromResult(res.IdList, nil, ws.FieldSort)
		}
	}()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, stale-while-revalidate=120")
	w.Header().Set("Access-Control-Allow-Origin", Origin)
	w.Header().Set("Age", "0")
	w.WriteHeader(http.StatusOK)

	result := SearchResponse{
		Items:     <-itemsChan,
		Facets:    <-facetsChan,
		Page:      page,
		PageSize:  pageSize,
		TotalHits: len(*searchResults),
	}

	err := json.NewEncoder(w).Encode(result)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

func (ws *WebServer) GetValues(w http.ResponseWriter, r *http.Request) {
	categories := removeEmptyStrings(strings.Split(strings.TrimPrefix(r.URL.Path, "/values/"), "/"))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, stale-while-revalidate=120")
	w.Header().Set("Access-Control-Allow-Origin", Origin)
	w.Header().Set("Age", "0")
	w.WriteHeader(http.StatusOK)
	if len(categories) == 0 {
		encErr := json.NewEncoder(w).Encode(ws.Index.KeyFacets[10].GetValues())
		if encErr != nil {
			http.Error(w, encErr.Error(), http.StatusInternalServerError)
		}
		return
	}

	sr := SearchRequest{
		Filters: index.Filters{
			StringFilter: []index.StringSearch{
				{
					Id:    10,
					Value: categories[0],
				},
			},
		},
	}
	for i := 1; i < len(categories); i++ {
		sr.Filters.StringFilter = append(sr.Filters.StringFilter, index.StringSearch{
			Id:    uint(10 + i),
			Value: categories[i],
		})
	}
	resultIds := ws.Index.Match(&sr.Filters, nil)
	values := map[string]bool{}
	for id := range *resultIds {
		item, ok := ws.Index.Items[id]
		if ok {
			if item.Fields != nil {
				for _, field := range item.Fields {
					if field.Id == uint(10+len(categories)) {
						values[field.Value] = true
					}
				}
			}
		}
	}
	valuesList := make([]string, len(values))
	i := 0
	for value := range values {
		valuesList[i] = value
		i++
	}

	encErr := json.NewEncoder(w).Encode(valuesList)
	if encErr != nil {
		http.Error(w, encErr.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) TrackClick(w http.ResponseWriter, r *http.Request) {
	session_id := handleSessionCookie(ws.Tracking, w, r)
	id := r.URL.Query().Get("id")
	itemId, err := strconv.Atoi(id)
	pos := r.URL.Query().Get("pos")
	position, _ := strconv.Atoi(pos)

	if ws.Tracking != nil && err == nil {
		err := ws.Tracking.TrackClick(uint32(session_id), uint(itemId), float32(position)/100.0)
		if err != nil {
			fmt.Printf("Failed to track click %v", err)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, stale-while-revalidate=2")
	w.Header().Set("Access-Control-Allow-Origin", Origin)
	w.Header().Set("Age", "0")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (ws *WebServer) StartServer(enableProfiling bool) error {

	srv := http.NewServeMux()

	srv.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv.HandleFunc("/api/filter", ws.Search)
	srv.HandleFunc("/api/suggest", ws.Suggest)
	srv.HandleFunc("/api/search", ws.QueryIndex)
	srv.HandleFunc("/track/click", ws.TrackClick)
	srv.HandleFunc("/admin/add", ws.AddItem)
	srv.HandleFunc("/admin/save", ws.Save)
	srv.HandleFunc("/api/values/", ws.GetValues)
	if enableProfiling {
		srv.HandleFunc("/debug/pprof/", pprof.Index)
		srv.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		srv.HandleFunc("/debug/pprof/profile", pprof.Profile)
		srv.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		srv.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
	return http.ListenAndServe(ws.ListenAddress, srv)
}
