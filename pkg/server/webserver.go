package server

import (
	"encoding/json"
	"net/http"
	"net/http/pprof"
	"strconv"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/persistance"
)

type WebServer struct {
	Index            *index.Index
	Db               *persistance.Persistance
	DefaultSort      *facet.SortIndex
	FieldSort        *facet.SortIndex
	FacetLimit       int
	SearchFacetLimit int
	ListenAddress    string
}

func (ws *WebServer) Search(w http.ResponseWriter, r *http.Request) {

	sr, err := QueryFromRequest(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	itemsChan := make(chan []index.ResultItem)
	facetsChan := make(chan index.Facets)
	defer close(itemsChan)
	defer close(facetsChan)
	matching := ws.Index.Match(&sr.Filters)

	totalHits := len(*matching)
	go func() {
		itemsChan <- ws.Index.GetItems(matching.SortedIds(ws.DefaultSort, sr.PageSize*(sr.Page+1)), sr.Page, sr.PageSize)
	}()
	go func() {
		if totalHits > ws.FacetLimit {
			facetsChan <- ws.Index.DefaultFacets
		} else {
			facetsChan <- ws.Index.GetFacetsFromResult(matching, &sr.Filters, ws.FieldSort)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, stale-while-revalidate=120")
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
	w.Header().Set("Cache-Control", "public, stale-while-revalidate=120")
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

	res := searchResults.ToResultWithSort(ws.DefaultSort)
	go func() {
		ids := res.SortIndex.SortMap(res.IdList, (page+1)*pageSize)
		itemsChan <- ws.Index.GetItems(ids, page, pageSize)
	}()
	go func() {

		//if len(searchResults) > ws.SearchFacetLimit {
		facetsChan <- index.Facets{}
		// } else {
		// 	facetsChan <- ws.Index.GetFacetsFromResult(&res.IdList, nil, ws.FieldSort)
		// }
	}()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, stale-while-revalidate=120")
	w.Header().Set("Access-Control-Allow-Origin", Origin)
	w.Header().Set("Age", "0")
	w.WriteHeader(http.StatusOK)

	result := SearchResponse{
		Items:     <-itemsChan,
		Facets:    <-facetsChan,
		Page:      page,
		PageSize:  pageSize,
		TotalHits: len(searchResults),
	}

	err := json.NewEncoder(w).Encode(result)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) StartServer(enableProfiling bool) error {

	srv := http.NewServeMux()

	srv.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv.HandleFunc("/filter", ws.Search)
	srv.HandleFunc("/suggest", ws.Suggest)
	srv.HandleFunc("/search", ws.QueryIndex)
	srv.HandleFunc("/add", ws.AddItem)
	srv.HandleFunc("/save", ws.Save)
	if enableProfiling {
		srv.HandleFunc("/debug/pprof/", pprof.Index)
		srv.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		srv.HandleFunc("/debug/pprof/profile", pprof.Profile)
		srv.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		srv.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
	return http.ListenAndServe(ws.ListenAddress, srv)
}
