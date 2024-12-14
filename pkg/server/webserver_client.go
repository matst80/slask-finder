package server

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/matst80/slask-finder/pkg/common"
	"github.com/matst80/slask-finder/pkg/embeddings"
	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/search"
	"github.com/matst80/slask-finder/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type searchResult struct {
	matching      *types.ItemList
	sort          *types.ByValue
	sortOverrides []index.SortOverride
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

func (ws *WebServer) getInitialIds(sr *FacetRequest) (*types.ItemList, *search.DocumentResult) {
	var initialIds *types.ItemList = nil
	var documentResult *search.DocumentResult = nil
	if sr.Query != "" {
		queryResult := ws.Index.Search.Search(sr.Query)
		initialIds = queryResult.ToResult()
		//initialIds = types.Intersect(types.ItemList{}, *queryResult)
		documentResult = queryResult
	}

	if len(sr.Stock) > 0 {
		resultStockIds := types.ItemList{}
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

	return initialIds, documentResult
}

func (ws *WebServer) getMatchAndSort(sr *SearchRequest, result chan<- searchResult) {
	matchingChan := make(chan *types.ItemList)
	sortChan := make(chan *types.ByValue)
	go noSearches.Inc()

	defer close(matchingChan)
	defer close(sortChan)

	initialIds, documentResult := ws.getInitialIds(sr.FacetRequest)
	isPopular := sr.Sort == "popular" || sr.Sort == ""

	if sr.Query != "" || isPopular {
		go func() {
			sortChan <- nil
		}()
	} else {
		go ws.Sorting.GetSorting(sr.Sort, sortChan)
	}

	go ws.Index.Match(sr.Filters, initialIds, matchingChan)

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

func makeBaseFacetRequest() *FacetRequest {
	return &FacetRequest{
		Filters: &index.Filters{
			StringFilter: []facet.StringFilter{},
			RangeFilter:  []facet.RangeFilter{},
		},
		Stock: []string{},
		Query: "",
	}
}

func makeBaseSearchRequest() *SearchRequest {
	return &SearchRequest{
		FacetRequest: makeBaseFacetRequest(),
		Sort:         "popular",
		Page:         0,
		PageSize:     40,
	}
}

type cacheWriter struct {
	key      string
	duration time.Duration
	store    func(string, []byte, time.Duration) error
}

func (cw *cacheWriter) Write(p []byte) (n int, err error) {
	cw.store(cw.key, p, cw.duration)
	return len(p), nil
}

func MakeCacheWriter(w io.Writer, key string, setRaw func(string, []byte, time.Duration) error) io.Writer {

	cacheWriter := &cacheWriter{
		key:      key,
		duration: time.Second * (60 * 5),
		store:    setRaw,
	}

	return io.MultiWriter(w, cacheWriter)

}

func (ws *WebServer) ContentSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	query = strings.TrimSpace(query)
	res := ws.ContentIndex.MatchQuery(query)
	defaultHeaders(w, r, true, "120")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.Encode(res)
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
		for keyId, sourceIds := range field.Keys {
			count = 0
			for id := range sourceIds {
				if _, ok := matchIds[id]; ok {
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

func (ws *WebServer) getSearchedFacets(baseIds *types.ItemList, filters *index.Filters, ch chan *index.JsonFacet, wg *sync.WaitGroup) {
	for _, s := range filters.StringFilter {
		if f, ok := ws.Index.Facets[s.Id]; ok {
			if !f.GetBaseField().HideFacet {
				wg.Add(1)
				go func(otherFilters *index.Filters) {
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
			if ok {
				wg.Add(1)
				go func(otherFilters *index.Filters) {
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
}

func (ws *WebServer) getOtherFacets(baseIds *types.ItemList, filters *index.Filters, ch chan *index.JsonFacet, wg *sync.WaitGroup) {

	fieldIds := make(map[uint]struct{})
	if len(*baseIds) > 65535 {
		for id := range ws.Index.Facets {
			fieldIds[id] = struct{}{}
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

				wg.Add(1)
				go getFacetResult(f, baseIds, ch, wg, func(facet *index.JsonFacet) *index.JsonFacet {
					if facet != nil && facet.CategoryLevel == 0 && !facet.Result.HasValues() {
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

func (ws *WebServer) GetFacets(w http.ResponseWriter, r *http.Request) {
	sr := makeBaseFacetRequest()
	err := GetFacetQueryFromRequest(r, sr)
	writer := io.Writer(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	go facetSearches.Inc()
	matchIds := make(chan *types.ItemList)
	defer close(matchIds)
	baseIds, _ := ws.getInitialIds(sr)
	go ws.Index.Match(sr.Filters, baseIds, matchIds)

	ch := make(chan *index.JsonFacet)
	wg := &sync.WaitGroup{}

	ids := <-matchIds
	ws.getOtherFacets(ids, sr.Filters, ch, wg)
	ws.getSearchedFacets(baseIds, sr.Filters, ch, wg)

	ret := make(map[uint]*index.JsonFacet)
	go func() {
		wg.Wait()
		close(ch)
	}()
	for facet := range ch {
		if facet != nil {
			ret[facet.Id] = facet
		}
	}
	defaultHeaders(w, r, true, "60")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(writer)
	for _, v := range *ws.Sorting.FieldSort {
		if d, ok := ret[v.Id]; ok {
			enc.Encode(d)
		}
	}

}

func (ws *WebServer) GetIds(w http.ResponseWriter, r *http.Request) {
	sr := makeBaseSearchRequest()
	err := GetQueryFromRequest(r, sr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resultChan := make(chan searchResult)

	defer close(resultChan)

	go ws.getMatchAndSort(sr, resultChan)

	result := <-resultChan

	defaultHeaders(w, r, false, "20")
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	encErr := enc.Encode(result.matching)
	if encErr != nil {
		http.Error(w, encErr.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) SearchStreamed(w http.ResponseWriter, r *http.Request) {

	sr := makeBaseSearchRequest()
	err := GetQueryFromRequest(r, sr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if sr.Sort == "" {
		sr.Sort = "popular"
	}

	session_id := common.HandleSessionCookie(ws.Tracking, w, r)
	if ws.Tracking != nil {
		go func() {
			err := ws.Tracking.TrackSearch(session_id, sr.Filters, sr.Query, sr.Page)
			if err != nil {
				fmt.Printf("Failed to track search %v", err)
			}
		}()
	}

	resultChan := make(chan searchResult)

	defer close(resultChan)

	go ws.getMatchAndSort(sr, resultChan)

	defaultHeaders(w, r, false, "3600")
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	start := sr.PageSize * sr.Page
	end := start + sr.PageSize
	result := <-resultChan
	sortedItemsChan := make(chan iter.Seq[*types.Item])
	go ws.Sorting.GetSortedItemsIterator(session_id, result.sort, result.matching, start, sortedItemsChan, result.sortOverrides...)

	fn := <-sortedItemsChan
	idx := 0
	for item := range fn {
		idx++
		enc.Encode(item)
		if idx >= end {
			break
		}
	}

	// var sortedIds iter.Seq[uint]
	// if sr.UseStaticPosition() {
	// 	sortedIds = result.sort.SortMapWithStaticPositions(*result.matching, ws.Sorting.GetStaticPositions())
	// } else {
	// 	sortedIds = result.sort.SortMap(*result.matching)
	// }
	// idx := 0
	// for id := range sortedIds {
	// 	idx++
	// 	if idx < start {
	// 		continue
	// 	}

	// 	if item, ok := ws.Index.Items[id]; ok {
	// 		enc.Encode(item)
	// 	}

	// 	if idx >= end {
	// 		break
	// 	}
	// }
	w.Write([]byte("\n"))

	enc.Encode(SearchResponse{
		Page:      sr.Page,
		PageSize:  sr.PageSize,
		Start:     start,
		End:       end,
		TotalHits: len(*result.matching),
		Sort:      sr.Sort,
	})
}

func (ws *WebServer) Suggest(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	query = strings.TrimSpace(query)
	words := strings.Split(query, " ")
	results := types.ItemList{}
	lastWord := words[len(words)-1]
	hasMoreWords := len(words) > 1
	other := words[:len(words)-1]
	go noSuggests.Inc()
	session_id := common.HandleSessionCookie(ws.Tracking, w, r)

	wordMatchesChan := make(chan []search.Match)
	sortChan := make(chan *types.ByValue)
	defer close(wordMatchesChan)
	defer close(sortChan)
	var docResult *search.DocumentResult = nil
	if hasMoreWords {
		docResult = ws.Index.Search.Search(query)
		types.Merge(results, *docResult)
		//results = *docResult
	}

	go ws.Index.AutoSuggest.FindMatchesForWord(lastWord, wordMatchesChan)

	defaultHeaders(w, r, false, "360")

	w.WriteHeader(http.StatusOK)

	sitem := &SuggestResult{}
	sitem.Other = other
	enc := json.NewEncoder(w)
	for _, s := range <-wordMatchesChan {
		sitem.Word = s.Word
		sitem.Hits = len(*s.Items)
		if !hasMoreWords || results.HasIntersection(s.Items) {
			json.NewEncoder(w).Encode(sitem)
			results.Merge(s.Items)
		}
	}
	// if hasMoreWords && docResult != nil {
	// 	go docResult.GetSortingWithAdditionalItems(&results, nil, sortChan)

	// } else {
	// 	go ws.Sorting.GetSorting("popular", sortChan)
	// }
	w.Write([]byte("\n"))

	sortedItemsChan := make(chan iter.Seq[*types.Item])
	if docResult != nil {
		o := index.SortOverride(*docResult)
		go ws.Sorting.GetSortedItemsIterator(session_id, nil, &results, 0, sortedItemsChan, o)
	} else {
		go ws.Sorting.GetSortedItemsIterator(session_id, nil, &results, 0, sortedItemsChan)
	}

	fn := <-sortedItemsChan
	idx := 0
	for item := range fn {
		idx++
		enc.Encode(item)
		if idx >= 20 {
			break
		}
	}

	// idx := 0
	// sort := <-sortChan
	// for id := range sort.SortMap(results) {
	// 	item, ok := ws.Index.Items[id]
	// 	if ok {

	// 		enc.Encode(item)
	// 		idx++
	// 		if idx > 20 {
	// 			break
	// 		}
	// 	}

	// }
	ws.Index.Lock()
	defer ws.Index.Unlock()
	ch := make(chan *index.JsonFacet)
	wg := &sync.WaitGroup{}

	ws.getOtherFacets(&results, &index.Filters{}, ch, wg)

	w.Write([]byte("\n"))

	ret := make(map[uint]*index.JsonFacet)
	go func() {
		wg.Wait()
		close(ch)
	}()
	for jsonFacet := range ch {
		if jsonFacet != nil {
			ret[jsonFacet.Id] = jsonFacet
		}
	}

	for _, v := range *ws.Sorting.FieldSort {
		if d, ok := ret[v.Id]; ok {
			_ = enc.Encode(d)
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
	var base *types.BaseField
	for _, field := range ws.Index.Facets {
		base = field.GetBaseField()
		if base.Id == uint(id) {
			defaultHeaders(w, r, true, "120")
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
	publicHeaders(w, r, true, "1200")

	w.WriteHeader(http.StatusOK)

	res := make([]types.BaseField, len(ws.Index.Facets))
	idx := 0
	for _, f := range ws.Index.Facets {
		res[idx] = *f.GetBaseField()
		idx++
	}

	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// type FieldSize struct {
// 	Id   uint `json:"id"`
// 	Size int  `json:"size"`
// }

// func (ws *WebServer) FacetSize(w http.ResponseWriter, r *http.Request) {
// 	publicHeaders(w, true, "1200")

// 	w.WriteHeader(http.StatusOK)

// 	res := make([]FieldSize, len(ws.Index.Facets))
// 	idx := 0
// 	for _, f := range ws.Index.Facets {
// 		res[idx] = FieldSize{
// 			Id:   f.GetBaseField().Id,
// 			Size: f.Size(),
// 		}
// 		idx++
// 	}

// 	err := json.NewEncoder(w).Encode(res)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 	}
// }

func (ws *WebServer) Related(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")
	id, err := strconv.Atoi(idString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	relatedChan := make(chan *types.ItemList)
	defer close(relatedChan)
	sortChan := make(chan *types.ByValue)
	defer close(sortChan)

	publicHeaders(w, r, false, "600")
	w.WriteHeader(http.StatusOK)
	go func(ch chan *types.ItemList) {
		related, err := ws.Index.Related(uint(id))
		if err != nil {
			ch <- &types.ItemList{}
			return
		}
		ch <- related
	}(relatedChan)
	go ws.Sorting.GetSorting("popular", sortChan)

	i := 0
	enc := json.NewEncoder(w)
	ws.Index.Lock()
	defer ws.Index.Unlock()
	related := <-relatedChan
	sort := <-sortChan
	for relatedId := range sort.SortMap(*related) {

		item, ok := ws.Index.Items[relatedId]
		if ok && (*item).GetId() != uint(id) {
			enc.Encode(item)
			i++
		}
		if i > 20 {
			break
		}
	}
}

type CategoryResult struct {
	Value    string            `json:"value"`
	Children []*CategoryResult `json:"children,omitempty"`
}

func CategoryResultFrom(c *index.Category) *CategoryResult {
	ret := &CategoryResult{}
	ret.Value = *c.Value
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
	publicHeaders(w, r, true, "600")
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

func (ws *WebServer) GetItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	itemId, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	item, ok := ws.Index.Items[uint(itemId)]
	if !ok {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	publicHeaders(w, r, true, "120")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) GetItems(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "600")
	items := make([]uint, 0)
	err := json.NewDecoder(r.Body).Decode(&items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result := make([]interface{}, len(items))
	i := 0
	for _, id := range items {
		item, ok := ws.Index.Items[id]
		if ok {
			result[i] = item
			i++
		}
	}
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(result[:i])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) SearchEmbeddings(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	query = strings.TrimSpace(query)
	typeField, ok := ws.Index.Facets[31158]
	if !ok {
		http.Error(w, "no type", http.StatusNotImplemented)
		return
	}
	values := typeField.GetValues()

	var productType string
	for _, ivalue := range values {
		value := ivalue.(string)

		if strings.Contains(query, strings.ToLower(value)) {
			productType = value
			break
		}

	}

	embeddings := embeddings.GetEmbedding(query)
	if ws.Embeddings == nil {
		http.Error(w, "Embeddings not enabled", http.StatusNotImplemented)
		return
	}
	results := ws.Embeddings.FindMatches(embeddings)
	toMatch := typeField.Match(productType)
	results.Ids.Intersect(*toMatch)
	defaultHeaders(w, r, true, "120")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	idx := 0
	for id := range results.SortIndex.SortMap(*toMatch) {
		item, ok := ws.Index.Items[id]
		if ok {
			enc.Encode(item)
		}
		idx++
		if idx > 40 {
			break
		}
	}

}

func (ws *WebServer) ClientHandler() *http.ServeMux {

	srv := http.NewServeMux()

	srv.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		defaultHeaders(w, r, false, "0")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	srv.HandleFunc("/content", ws.ContentSearch)
	srv.HandleFunc("/facets", ws.GetFacets)
	srv.HandleFunc("/ai-search", ws.SearchEmbeddings)
	srv.HandleFunc("/related/{id}", ws.Related)
	srv.HandleFunc("/facet-list", ws.Facets)
	srv.HandleFunc("/suggest", ws.Suggest)
	srv.HandleFunc("/categories", ws.Categories)
	//srv.HandleFunc("/search", ws.QueryIndex)
	srv.HandleFunc("/stream", ws.SearchStreamed)

	srv.HandleFunc("/ids", ws.GetIds)
	srv.HandleFunc("GET /get/{id}", ws.GetItem)
	srv.HandleFunc("POST /get", ws.GetItems)
	srv.HandleFunc("/values/{id}", ws.GetValues)

	return srv
}
