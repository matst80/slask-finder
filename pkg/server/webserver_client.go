package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"tornberg.me/facet-search/pkg/common"
	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/search"
	"tornberg.me/facet-search/pkg/tracking"
)

type searchResult struct {
	matching *facet.IdList
	sort     *facet.SortIndex
}

var (
	noSearches = promauto.NewCounter(prometheus.CounterOpts{
		Name: "slaskfinder_searches_total",
		Help: "The total number of processed searches",
	})
	noSuggests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "slaskfinder_suggest_total",
		Help: "The total number of processed suggestions",
	})
	// avgSearchTime = promauto.NewCounter(prometheus.CounterOpts{
	// 	Name: "slaskfinder_searches_ms_total",
	// 	Help: "The total number of ms consumed by search",
	// })
	facetSearches = promauto.NewCounter(prometheus.CounterOpts{
		Name: "slaskfinder_facets_total",
		Help: "The total number of processed searches",
	})
	// facetGenerationTime = promauto.NewCounter(prometheus.CounterOpts{
	// 	Name: "slaskfinder_searches_ms_total",
	// 	Help: "The total number of ms consumed by search",
	// })
)

func (ws *WebServer) getMatchAndSort(sr *SearchRequest, result chan<- searchResult) {
	matchingChan := make(chan *facet.IdList)
	sortChan := make(chan *facet.SortIndex)
	go noSearches.Inc()

	defer close(matchingChan)
	defer close(sortChan)

	var initialIds *facet.IdList = nil

	if sr.Query != "" {
		queryResult := ws.Index.Search.Search(sr.Query)

		initialIds = queryResult.ToResult()
		if sr.Sort == "popular" || sr.Sort == "" {
			go queryResult.GetSorting(sortChan)
		} else {
			go ws.Sorting.GetSorting(sr.Sort, sortChan)
		}
	} else {
		go ws.Sorting.GetSorting(sr.Sort, sortChan)
	}

	if len(sr.Stock) > 0 {
		resultStockIds := facet.IdList{}
		for _, stockId := range sr.Stock {
			stockIds, ok := ws.Index.ItemsInStock[stockId]
			if ok {
				resultStockIds.Merge(&stockIds)
			}
		}

		if initialIds == nil {
			initialIds = &resultStockIds
		} else {
			initialIds.Intersect(resultStockIds)
		}

	}

	go ws.Index.Match(&sr.Filters, initialIds, matchingChan)

	result <- searchResult{
		matching: <-matchingChan,
		sort:     <-sortChan,
	}

}

func (ws *WebServer) Search(w http.ResponseWriter, r *http.Request) {

	sr, err := QueryFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	facetsChan := make(chan index.Facets)
	resultChan := make(chan searchResult)

	defer close(facetsChan)
	defer close(resultChan)

	go ws.getMatchAndSort(&sr, resultChan)

	result := <-resultChan
	totalHits := len(*result.matching)

	go getFacetsForIds(result.matching, ws.Index, &sr.Filters, ws.Sorting.FieldSort, facetsChan)
	go facetSearches.Inc()
	defaultHeaders(w, true, "20")
	enc := json.NewEncoder(w)
	w.WriteHeader(http.StatusOK)

	data := SearchResponse{
		Facets:    <-facetsChan,
		TotalHits: totalHits,
	}

	encErr := enc.Encode(data)
	if encErr != nil {
		http.Error(w, encErr.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) GetIds(w http.ResponseWriter, r *http.Request) {
	sr, err := QueryFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resultChan := make(chan searchResult)

	defer close(resultChan)

	go ws.getMatchAndSort(&sr, resultChan)

	result := <-resultChan

	defaultHeaders(w, false, "20")
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	encErr := enc.Encode(result.matching)
	if encErr != nil {
		http.Error(w, encErr.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) SearchStreamed(w http.ResponseWriter, r *http.Request) {

	sr, err := QueryFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	session_id := common.HandleSessionCookie(ws.Tracking, w, r)
	go func() {
		if ws.Tracking != nil {
			err := ws.Tracking.TrackSearch(uint32(session_id), &sr.Filters, sr.Query, sr.Page)
			if err != nil {
				fmt.Printf("Failed to track search %v", err)
			}
		}
	}()

	resultChan := make(chan searchResult)

	defer close(resultChan)

	go ws.getMatchAndSort(&sr, resultChan)

	defaultHeaders(w, false, "20")
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	start := sr.PageSize * sr.Page
	end := start + sr.PageSize
	result := <-resultChan

	ritem := &index.ResultItem{}
	for idx, id := range (*result.matching).SortedIdsWithStaticPositions(result.sort, ws.Sorting.GetStaticPositions(), end) {
		if idx < start {
			continue
		}
		item, ok := ws.Index.Items[id]
		if ok {
			index.ToResultItem(item, ritem)
			enc.Encode(ritem)
		}
	}
	//w.Write([]byte("\n"))
}

func (ws *WebServer) Suggest(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	query = strings.TrimSpace(query)
	words := strings.Split(query, " ")
	results := facet.IdList{}
	lastWord := words[len(words)-1]
	hasMoreWords := len(words) > 1
	other := words[:len(words)-1]
	go noSuggests.Inc()

	wordMatchesChan := make(chan []search.Match)
	sortChan := make(chan *facet.SortIndex)
	defer close(wordMatchesChan)
	defer close(sortChan)
	var docResult *search.DocumentResult = nil
	if hasMoreWords {
		docResult = ws.Index.Search.Search(query)
		results = *docResult.ToResult()
	}

	go ws.Index.AutoSuggest.FindMatchesForWord(lastWord, wordMatchesChan)

	defaultHeaders(w, false, "360")

	w.WriteHeader(http.StatusOK)

	sitem := &SuggestResult{}
	sitem.Other = other
	enc := json.NewEncoder(w)
	for _, s := range <-wordMatchesChan {

		sitem.Word = s.Word
		sitem.Hits = len(*s.Ids)
		if !hasMoreWords || results.HasIntersection(s.Ids) {
			json.NewEncoder(w).Encode(sitem)
			results.Merge(s.Ids)
		}
	}
	if hasMoreWords && docResult != nil {
		go docResult.GetSortingWithAdditionalItems(&results, ws.Index.Search.BaseSortMap, sortChan)

	} else {
		go ws.Sorting.GetSorting("popular", sortChan)
	}
	w.Write([]byte("\n"))
	ritem := &index.ResultItem{}

	for _, id := range results.SortedIds(<-sortChan, 40) {
		item, ok := ws.Index.Items[id]
		if ok {
			index.ToResultItem(item, ritem)
			enc.Encode(ritem)
		}
	}
	ws.Index.Lock()
	defer ws.Index.Unlock()
	facets := ws.Index.GetFacetsFromResult(&results, &index.Filters{}, ws.Sorting.FieldSort)
	w.Write([]byte("\n"))
	enc.Encode(facets)
}

func (ws *WebServer) Learn(w http.ResponseWriter, r *http.Request) {
	fieldStrings := strings.Split(r.URL.Query().Get("fields"), ",")
	fields := make([]int, len(fieldStrings))
	for i, fieldString := range fieldStrings {
		field, fieldError := strconv.Atoi(fieldString)
		if fieldError != nil {
			http.Error(w, fieldError.Error(), http.StatusBadRequest)
			return
		}
		fields[i] = field
	}

	categories := removeEmptyStrings(strings.Split(strings.TrimPrefix(r.URL.Path, "/api/learn/"), "/"))
	w.WriteHeader(http.StatusOK)
	baseSearch := SearchRequest{
		PageSize: 10000,
		Page:     0,
	}
	resultIds := ws.getCategoryItemIds(categories, &baseSearch, 10)

	for id := range *resultIds {
		item, ok := ws.Index.Items[id]
		if ok {
			parts := make([]string, len(fields)+1)
			parts[0] = item.Sku
			for i, field := range fields {

				for _, itemField := range item.IntegerFields {
					if itemField.Id == uint(field) {
						parts[i+1] = strconv.Itoa(itemField.Value)
						break
					}
				}
				if parts[i+1] == "" {
					for _, itemField := range item.DecimalFields {
						if itemField.Id == uint(field) {
							parts[i+1] = fmt.Sprintf("%v", itemField.Value)
							break
						}
					}
				}
			}

			fmt.Fprintln(w, strings.Join(parts, ";"))
		}
	}
}

func (ws *WebServer) GetValues(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")
	id, err := strconv.Atoi(idString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ws.Index.Lock()
	defer ws.Index.Unlock()
	for _, field := range ws.Index.KeyFacets {
		if field.BaseField.Id == uint(id) {
			defaultHeaders(w, true, "120")
			w.WriteHeader(http.StatusOK)
			err := json.NewEncoder(w).Encode(field.GetValues())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

func (ws *WebServer) Facets(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, true, "1200")

	w.WriteHeader(http.StatusOK)

	res := make([]FacetItem, len(ws.Index.KeyFacets)+len(ws.Index.DecimalFacets)+len(ws.Index.IntFacets))
	i := 0
	for _, f := range ws.Index.KeyFacets {
		res[i] = FacetItem{
			BaseField: f.BaseField,
			FieldType: "key",
			Count:     f.UniqueCount(),
		}
		i++
	}
	for _, f := range ws.Index.DecimalFacets {
		res[i] = FacetItem{
			BaseField: f.BaseField,
			FieldType: "decimal",
			Count:     int(f.Max - f.Min),
		}
		i++
	}
	for _, f := range ws.Index.IntFacets {
		res[i] = FacetItem{
			BaseField: f.BaseField,
			FieldType: "int",
			Count:     int(f.Max - f.Min),
		}
		i++
	}

	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) Related(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")
	id, err := strconv.Atoi(idString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defaultHeaders(w, false, "120")
	w.WriteHeader(http.StatusOK)
	related, err := ws.Index.Related(uint(id))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	sort := ws.Index.Sorting.GetSort("popular")

	ritem := &index.ResultItem{}
	i := 0
	enc := json.NewEncoder(w)
	ws.Index.Lock()
	defer ws.Index.Unlock()
	for _, relatedId := range (*related).SortedIds(sort, len(*related)) {

		item, ok := ws.Index.Items[relatedId]
		if ok && i < 20 && item.Id != uint(id) {
			index.ToResultItem(item, ritem)
			enc.Encode(ritem)
			i++
		}
	}

}

func (ws *WebServer) TrackClick(w http.ResponseWriter, r *http.Request) {
	session_id := common.HandleSessionCookie(ws.Tracking, w, r)
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
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (ws *WebServer) TrackImpression(w http.ResponseWriter, r *http.Request) {
	session_id := common.HandleSessionCookie(ws.Tracking, w, r)
	data := make([]tracking.Impression, 0)
	err := json.NewDecoder(r.Body).Decode(&data)

	if ws.Tracking != nil && err == nil {

		err := ws.Tracking.TrackImpressions(uint32(session_id), data)
		if err != nil {
			fmt.Printf("Failed to track click %v", err)
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (ws *WebServer) TrackAction(w http.ResponseWriter, r *http.Request) {
	session_id := common.HandleSessionCookie(ws.Tracking, w, r)
	var data tracking.TrackingAction
	err := json.NewDecoder(r.Body).Decode(&data)

	if ws.Tracking != nil && err == nil {

		err := ws.Tracking.TrackAction(uint32(session_id), data)
		if err != nil {
			fmt.Printf("Failed to track click %v", err)
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

type CategoryResult struct {
	Value    string            `json:"value"`
	Children []*CategoryResult `json:"children,omitempty"`
}

func CategoryResultFrom(c *index.Category) *CategoryResult {
	ret := &CategoryResult{}
	ret.Value = c.Value
	ret.Children = make([]*CategoryResult, 0)
	if c.Children != nil {
		for _, child := range c.Children {
			if child != nil {
				ret.Children = append(ret.Children, CategoryResultFrom(child))
			}
		}
	}
	return ret
}

func (ws *WebServer) Categories(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, true, "120")
	w.WriteHeader(http.StatusOK)
	categories := ws.Index.GetCategories()
	result := make([]*CategoryResult, 0)

	for _, category := range categories {
		if category != nil {
			result = append(result, CategoryResultFrom(category))
		}
	}

	err := json.NewEncoder(w).Encode(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) ClientHandler() *http.ServeMux {

	srv := http.NewServeMux()

	srv.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		defaultHeaders(w, false, "0")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv.HandleFunc("/filter", ws.Search)
	srv.HandleFunc("/learn/", ws.Learn)
	srv.HandleFunc("/related/{id}", ws.Related)
	srv.HandleFunc("/facet-list", ws.Facets)
	srv.HandleFunc("/suggest", ws.Suggest)
	srv.HandleFunc("/categories", ws.Categories)
	//srv.HandleFunc("/search", ws.QueryIndex)
	srv.HandleFunc("/stream", ws.SearchStreamed)
	srv.HandleFunc("/ids", ws.GetIds)
	srv.HandleFunc("/get/{id}", ws.GetItem)
	//srv.HandleFunc("/stream/facets", ws.FacetsStreamed)
	srv.HandleFunc("/values/{id}", ws.GetValues)
	srv.HandleFunc("/track/click", ws.TrackClick)
	srv.HandleFunc("/track/impressions", ws.TrackImpression)
	srv.HandleFunc("/track/action", ws.TrackAction)

	return srv
}
