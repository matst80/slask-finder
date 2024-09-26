package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
)

func (ws *WebServer) Search(w http.ResponseWriter, r *http.Request) {
	session_id := handleSessionCookie(ws.Tracking, w, r)
	sr, err := QueryFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//cacheHelper := NewCacheHelper[index.Facets](ws.Cache)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	itemsChan := make(chan []index.ResultItem)
	facetsChan := make(chan index.Facets)
	matchingChan := make(chan *facet.IdList)
	defer close(itemsChan)
	defer close(facetsChan)
	defer close(matchingChan)

	go ws.matchQuery(&sr, matchingChan)
	itemSort := ws.Sorting.GetSort(sr.Sort)

	matching := <-matchingChan
	totalHits := len(*matching)

	go getSortedItems(matching, ws.Index, itemSort, sr.Page, sr.PageSize, itemsChan)
	go getFacetsForIds(matching, ws.Index, &sr.Filters, ws.Sorting.FieldSort, facetsChan)

	go func() {
		if ws.Tracking != nil {
			err := ws.Tracking.TrackSearch(uint32(session_id), &sr.Filters, sr.Query)
			if err != nil {
				fmt.Printf("Failed to track search %v", err)
			}
		}
	}()
	defaultHeaders(w, true, "20")
	enc := json.NewEncoder(w)
	w.WriteHeader(http.StatusOK)

	data := SearchResponse{
		Items:     <-itemsChan,
		Facets:    <-facetsChan,
		Page:      sr.Page,
		PageSize:  sr.PageSize,
		TotalHits: totalHits,
	}

	encErr := enc.Encode(data)
	if encErr != nil {
		http.Error(w, encErr.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) FacetsStreamed(w http.ResponseWriter, r *http.Request) {
	sr, err := QueryFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defaultHeaders(w, true, "60")
	w.WriteHeader(http.StatusOK)

	facetsChan := make(chan index.Facets)
	matchingChan := make(chan *facet.IdList)
	defer close(facetsChan)
	defer close(matchingChan)
	go ws.matchQuery(&sr, matchingChan)
	go getFacetsForIds(<-matchingChan, ws.Index, &sr.Filters, ws.Sorting.FieldSort, facetsChan)
	enc := json.NewEncoder(w)
	encErr := enc.Encode(<-facetsChan)
	if encErr != nil {
		http.Error(w, encErr.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) SearchStreamed(w http.ResponseWriter, r *http.Request) {
	session_id := handleSessionCookie(ws.Tracking, w, r)
	sr, err := QueryFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	matchingChan := make(chan *facet.IdList)

	defer close(matchingChan)

	go ws.matchQuery(&sr, matchingChan)

	itemSort := ws.Sorting.GetSort(sr.Sort)

	go func() {
		if ws.Tracking != nil {
			err := ws.Tracking.TrackSearch(uint32(session_id), &sr.Filters, sr.Query)
			if err != nil {
				fmt.Printf("Failed to track search %v", err)
			}
		}
	}()
	defaultHeaders(w, false, "20")
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	start := sr.PageSize * sr.Page
	matching := <-matchingChan
	for idx, id := range matching.SortedIds(itemSort, sr.PageSize*(sr.Page+1)) {
		item, ok := ws.Index.Items[id]
		if ok && idx >= start {
			enc.Encode(index.MakeResultItem(item))
		}
	}
	w.Write([]byte("\n"))
}

func (ws *WebServer) Suggest(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	suggestions := ws.Index.AutoSuggest.FindMatches(query)
	defaultHeaders(w, true, "120")

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

	searchResults := ws.Index.Search.Search(query)
	res := searchResults.ToResultWithSort()
	go getSortedItems(res.IdList, ws.Index, ws.Sorting.DefaultSort, page, pageSize, itemsChan)
	go getFacetsForIds(res.IdList, ws.Index, nil, ws.Sorting.FieldSort, facetsChan)

	defaultHeaders(w, true, "360")
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
	categories := removeEmptyStrings(strings.Split(strings.TrimPrefix(r.URL.Path, "/api/values/"), "/"))
	defaultHeaders(w, true, "120")
	w.WriteHeader(http.StatusOK)
	if len(categories) == 0 {
		encErr := json.NewEncoder(w).Encode(ws.Index.KeyFacets[10].GetValues())
		if encErr != nil {
			http.Error(w, encErr.Error(), http.StatusInternalServerError)
		}
		return
	}
	baseSearch := SearchRequest{
		PageSize: 1000,
		Page:     0,
	}
	resultIds := ws.getCategoryItemIds(categories, &baseSearch, 10)

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

func (ws *WebServer) Facets(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, true, "1200")

	w.WriteHeader(http.StatusOK)

	res := make([]FacetItem, len(ws.Index.KeyFacets)+len(ws.Index.DecimalFacets)+len(ws.Index.IntFacets))
	i := 0
	for _, f := range ws.Index.KeyFacets {
		res[i] = FacetItem{
			Id:    f.Id,
			Name:  f.Name,
			Count: f.UniqueCount(),
		}
		i++
	}
	for _, f := range ws.Index.DecimalFacets {
		res[i] = FacetItem{
			Id:    f.Id,
			Name:  f.Name,
			Count: int(f.Max - f.Min),
		}
		i++
	}
	for _, f := range ws.Index.IntFacets {
		res[i] = FacetItem{
			Id:    f.Id,
			Name:  f.Name,
			Count: int(f.Max - f.Min),
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

	item, ok := ws.Index.Items[uint(id)]
	if !ok {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	defaultHeaders(w, true, "120")
	w.WriteHeader(http.StatusOK)
	// related := ws.Index.Search.Related(item)
	err = json.NewEncoder(w).Encode(item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	defaultHeaders(w, false, "2")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
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
	srv.HandleFunc("/search", ws.QueryIndex)
	srv.HandleFunc("/stream/items", ws.SearchStreamed)
	srv.HandleFunc("/stream/facets", ws.FacetsStreamed)
	srv.HandleFunc("/values/", ws.GetValues)
	srv.HandleFunc("/track/click", ws.TrackClick)

	return srv
}
