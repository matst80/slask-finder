package server

import (
	"encoding/json"
	"iter"
	"maps"
	"net/http"

	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/search"
	"github.com/matst80/slask-finder/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"strconv"
	"strings"
	"sync"
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
	facetSearches = promauto.NewCounter(prometheus.CounterOpts{
		Name: "slaskfinder_facets_total",
		Help: "The total number of processed searches",
	})
)

func (ws *WebServer) getInitialIds(sr *types.FacetRequest) *types.ItemList {
	var initialIds *types.ItemList = nil
	//var documentResult *search.DocumentResult = nil
	if sr.Query != "" {
		if sr.Query == "*" {
			// probably should copy this
			cloned := types.ItemList{}
			maps.Copy(cloned, ws.Index.All)
			initialIds = &cloned
		} else {
			initialIds = ws.Index.Search.Search(sr.Query)

			//documentResult = queryResult
		}
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

	return initialIds
}

func (ws *WebServer) ContentSearch(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	query := r.URL.Query().Get("q")
	query = strings.TrimSpace(query)
	res := ws.ContentIndex.MatchQuery(query)
	defaultHeaders(w, r, true, "160")
	w.WriteHeader(http.StatusOK)
	var err error
	for content := range res {
		err = enc.Encode(content)
	}
	return err
}

func (ws *WebServer) GetFacets(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {

	sr, err := GetFacetQueryFromRequest(r)
	if err != nil {
		return err
	}
	go facetSearches.Inc()

	matchIds := make(chan *types.ItemList)
	defer close(matchIds)
	baseIds := ws.getInitialIds(sr)
	go ws.Index.Match(sr.Filters, baseIds, matchIds)

	ch := make(chan *index.JsonFacet)
	wg := &sync.WaitGroup{}

	ids := <-matchIds
	ws.getOtherFacets(ids, sr, ch, wg)
	ws.getSearchedFacets(baseIds, sr.Filters, ch, wg)

	// todo optimize
	go func() {
		wg.Wait()
		close(ch)
	}()

	ret := make([]*index.JsonFacet, 0)
	for item := range ch {
		if item != nil {
			ret = append(ret, item)
		}
	}

	defaultHeaders(w, r, true, "60")
	w.WriteHeader(http.StatusOK)

	return enc.Encode(ws.Sorting.GetSortedFields(sessionId, ret))
}

func (ws *WebServer) GetIds(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {

	sr, err := GetQueryFromRequest(r)
	if err != nil {
		return err
	}

	resultChan := make(chan searchResult)

	defer close(resultChan)

	go ws.getMatchAndSort(sr, resultChan)

	result := <-resultChan

	defaultHeaders(w, r, false, "20")
	w.WriteHeader(http.StatusOK)

	return enc.Encode(result.matching)
}

func (ws *WebServer) SearchStreamed(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {

	sr, err := GetQueryFromRequest(r)
	if err != nil {
		return err
	}

	if ws.Tracking != nil && !sr.SkipTracking {
		go ws.Tracking.TrackSearch(sessionId, sr.Filters, sr.Query, sr.Page, r)
	}

	resultChan := make(chan searchResult)

	defer close(resultChan)

	go ws.getMatchAndSort(sr, resultChan)

	defaultHeaders(w, r, false, "10")
	w.WriteHeader(http.StatusOK)

	start := sr.PageSize * sr.Page
	end := start + sr.PageSize
	result := <-resultChan
	sortedItemsChan := make(chan iter.Seq[*types.Item])
	go ws.Sorting.GetSortedItemsIterator(sessionId, result.sort, result.matching, start, sortedItemsChan, result.sortOverrides...)

	fn := <-sortedItemsChan
	idx := 0

	for item := range fn {
		idx++
		err = enc.Encode(item)
		if err != nil {
			break
		}
		if idx >= end {
			break
		}
	}
	if err != nil {
		return err
	}

	_, err = w.Write([]byte("\n"))
	if err != nil {
		return err
	}

	return enc.Encode(SearchResponse{
		Page:      sr.Page,
		PageSize:  sr.PageSize,
		Start:     start,
		End:       end,
		TotalHits: len(*result.matching),
		Sort:      sr.Sort,
	})
}

func (ws *WebServer) Suggest(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {

	query := r.URL.Query().Get("q")
	if query == "" {
		items, _ := ws.Sorting.GetSessionData(uint(sessionId))
		sortedItems := items.ToSortedLookup()
		//sortedFields := fields.ToSortedLookup()
		defaultHeaders(w, r, true, "60")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("\n"))
		max := 30
		for _, v := range sortedItems {
			item, ok := ws.Index.Items[v.Id]
			if ok {
				err := enc.Encode(item)
				if err != nil {
					return err
				}
				max--
				if max <= 0 {
					break
				}
			}
		}
		w.Write([]byte("\n"))
		return nil
	}
	query = strings.TrimSpace(query)
	words := strings.Split(query, " ")
	results := types.ItemList{}
	lastWord := words[len(words)-1]
	hasMoreWords := len(words) > 1
	other := words[:len(words)-1]
	go noSuggests.Inc()

	wordMatchesChan := make(chan []search.Match)
	sortChan := make(chan *types.ByValue)
	defer close(wordMatchesChan)
	defer close(sortChan)
	var docResult *types.ItemList
	if hasMoreWords {
		docResult = ws.Index.Search.Search(query)
		types.Merge(results, *docResult)
		//results = *docResult
	}

	go ws.Index.Search.FindTrieMatchesForWord(lastWord, wordMatchesChan)

	defaultHeaders(w, r, false, "360")

	w.WriteHeader(http.StatusOK)

	suggestResult := &SuggestResult{}
	suggestResult.Other = other
	hasResults := len(results) > 0
	var err error
	for _, s := range <-wordMatchesChan {
		suggestResult.Prefix = lastWord
		suggestResult.Word = s.Word
		totalHits := len(*s.Items)
		if totalHits > 0 {
			if !hasResults {
				suggestResult.Hits = totalHits
				err = enc.Encode(suggestResult)
				results.Merge(s.Items)
			} else if results.HasIntersection(s.Items) {
				suggestResult.Hits = results.IntersectionLen(*s.Items)
				err = enc.Encode(suggestResult)
				// dont intersect with the other words yet since partial
				//results.Intersect(*s.Items)
			}
		}

	}
	if err != nil {
		return err
	}
	// if hasMoreWords && docResult != nil {
	// 	go docResult.GetSortingWithAdditionalItems(&results, nil, sortChan)

	// } else {
	// 	go ws.Sorting.GetSorting("popular", sortChan)
	// }
	_, err = w.Write([]byte("\n"))

	sortedItemsChan := make(chan iter.Seq[*types.Item])
	// if docResult != nil {
	// 	//o := index.SortOverride(docResult)
	// 	go ws.Sorting.GetSortedItemsIterator(sessionId, nil, &results, 0, sortedItemsChan, o)
	// } else {
	go ws.Sorting.GetSortedItemsIterator(sessionId, nil, &results, 0, sortedItemsChan)
	//}

	fn := <-sortedItemsChan
	idx := 0
	for item := range fn {
		idx++
		err = enc.Encode(item)
		if idx >= 20 || err != nil {
			break
		}
	}
	if err != nil {
		return err
	}

	ws.Index.Lock()
	defer ws.Index.Unlock()
	ch := make(chan *index.JsonFacet)
	wg := &sync.WaitGroup{}

	ws.getOtherFacets(&results, &types.FacetRequest{Filters: &types.Filters{}}, ch, wg)

	_, err = w.Write([]byte("\n"))

	//ret := make(map[uint]*index.JsonFacet)
	go func() {
		wg.Wait()
		close(ch)
	}()
	for jsonFacet := range ch {
		if jsonFacet != nil {
			err = enc.Encode(jsonFacet)
		}
	}

	// for _, v := range *ws.Sorting.FieldSort {
	// 	if d, ok := ret[v.Id]; ok {
	// 		err = enc.Encode(d)
	// 	}
	// }
	return err
}

func (ws *WebServer) GetValues(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	idString := r.PathValue("id")
	id, err := strconv.Atoi(idString)
	if err != nil {
		return err
	}
	ws.Index.Lock()
	defer ws.Index.Unlock()
	var base *types.BaseField
	for _, field := range ws.Index.Facets {
		base = field.GetBaseField()
		if base.Id == uint(id) {
			defaultHeaders(w, r, true, "120")
			w.WriteHeader(http.StatusOK)
			err := enc.Encode(field.GetValues())
			return err
		}
	}
	w.WriteHeader(http.StatusNotFound)
	return nil
}

func (ws *WebServer) Facets(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	publicHeaders(w, r, true, "1200")

	w.WriteHeader(http.StatusOK)

	res := make([]types.BaseField, len(ws.Index.Facets))
	idx := 0
	for _, f := range ws.Index.Facets {
		res[idx] = *f.GetBaseField()
		idx++
	}

	return enc.Encode(res)
}

func (ws *WebServer) Popular(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	items, _ := ws.Sorting.GetSessionData(uint(sessionId))
	sortedItems := items.ToSortedLookup()
	//sortedFields := fields.ToSortedLookup()
	defaultHeaders(w, r, true, "60")

	w.WriteHeader(http.StatusOK)
	for _, v := range sortedItems {
		item, ok := ws.Index.Items[v.Id]
		if ok {
			err := enc.Encode(item)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type Similar struct {
	ProductType string       `json:"productType"`
	Count       int          `json:"count"`
	Popularity  float64      `json:"popularity"`
	Items       []types.Item `json:"items"`
}

func (ws *WebServer) Similar(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	items, fields := ws.Sorting.GetSessionData(uint(sessionId))
	articleTypes := map[string]float64{}
	itemChan := make(chan *Similar)

	wg := &sync.WaitGroup{}
	pop := ws.Sorting.GetSort("popular")
	delete(*fields, 31158)
	for id := range *items {
		if item, ok := ws.Index.Items[id]; ok {
			if itemType, typeOk := (*item).GetFields()[31158]; typeOk {
				articleTypes[itemType.(string)]++
			}
		}
	}
	getSimilar := func(articleType string, ret chan *Similar, wg *sync.WaitGroup, sort *types.ByValue, popularity float64) {
		ids := make(chan *types.ItemList)
		defer close(ids)
		defer wg.Done()
		filter := types.Filters{
			StringFilter: []types.StringFilter{
				{Id: 31158, Value: articleType},
			},
		}

		go ws.Index.Match(&filter, nil, ids)
		resultIds := <-ids
		l := len(*resultIds)
		limit := min(l, 40)
		similar := Similar{
			ProductType: articleType,
			Count:       l,
			Popularity:  popularity,
			Items:       make([]types.Item, 0, limit),
		}
		for id := range sort.SortMap(*resultIds) {
			if _, found := (*items)[id]; found {
				continue
			}
			if item, ok := ws.Index.Items[id]; ok {
				similar.Items = append(similar.Items, *item)
				if len(similar.Items) >= limit {
					break
				}
			}
		}
		ret <- &similar
	}

	for i, typeValue := range index.ToSortedMap(articleTypes) {
		if i > 4 {
			break
		}
		wg.Add(1)
		go getSimilar(typeValue, itemChan, wg, pop, articleTypes[typeValue])
	}
	go func() {
		wg.Wait()
		close(itemChan)
	}()
	defaultHeaders(w, r, false, "600")
	w.WriteHeader(http.StatusOK)
	for similar := range itemChan {
		err := enc.Encode(similar)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ws *WebServer) Related(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {

	idString := r.PathValue("id")
	id, err := strconv.Atoi(idString)
	if err != nil {
		return err
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

	ws.Index.Lock()
	defer ws.Index.Unlock()
	related := <-relatedChan
	sort := <-sortChan
	for relatedId := range sort.SortMap(*related) {

		item, ok := ws.Index.Items[relatedId]
		if ok && (*item).GetId() != uint(id) {
			err = enc.Encode(item)
			i++
		}
		if i > 20 || err != nil {
			break
		}
	}
	return err
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

// func (ws *WebServer) Categories(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
// 	publicHeaders(w, r, true, "600")
// 	w.WriteHeader(http.StatusOK)
// 	categories := ws.Index.GetCategories()
// 	result := make([]*CategoryResult, 0)

// 	for _, category := range categories {
// 		if category != nil {
// 			result = append(result, CategoryResultFrom(category))
// 		}
// 	}

// 	return enc.Encode(result)
// }

func (ws *WebServer) GetItem(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	id := r.PathValue("id")
	itemId, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	item, ok := ws.Index.Items[uint(itemId)]
	if !ok {
		return err
	}
	publicHeaders(w, r, true, "120")
	w.WriteHeader(http.StatusOK)
	return enc.Encode(item)
}

func (ws *WebServer) GetItems(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	defaultHeaders(w, r, true, "600")
	items := make([]uint, 0)
	err := json.NewDecoder(r.Body).Decode(&items)
	if err != nil {
		return err
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
	return enc.Encode(result[:i])
}

// func (ws *WebServer) SearchEmbeddings(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
// 	query := r.URL.Query().Get("q")
// 	query = strings.TrimSpace(query)
// 	typeField, ok := ws.Index.Facets[31158]
// 	if !ok {
// 		return fmt.Errorf("facet not found")
// 	}
// 	values := typeField.GetValues()

// 	var productType string
// 	for _, valueInterface := range values {
// 		value := valueInterface.(string)

// 		if strings.Contains(query, strings.ToLower(value)) {
// 			productType = value
// 			break
// 		}

// 	}

// 	queryVector := embeddings.GetEmbedding(query)
// 	if ws.Embeddings == nil {
// 		return fmt.Errorf("embeddings not enabled")
// 	}
// 	results := ws.Embeddings.FindMatches(queryVector)
// 	toMatch := typeField.Match(productType)
// 	results.Ids.Intersect(*toMatch)
// 	defaultHeaders(w, r, true, "120")
// 	w.WriteHeader(http.StatusOK)
// 	var err error
// 	idx := 0
// 	for id := range results.SortIndex.SortMap(*toMatch) {
// 		item, ok := ws.Index.Items[id]
// 		if ok {
// 			err = enc.Encode(item)
// 		}
// 		idx++
// 		if idx > 40 || err != nil {
// 			break
// 		}
// 	}
// 	return err
// }

func (ws *WebServer) TriggerWords(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	defaultHeaders(w, r, true, "1200")
	ret := make(map[string]uint)
	for id, facet := range ws.Index.Facets {
		base := facet.GetBaseField()
		if (base.Type != "" || base.CategoryLevel > 0) && !base.HideFacet && facet.GetType() == types.FacetKeyType {
			for _, line := range facet.GetValues() {
				switch values := line.(type) {
				case []string:
					for _, v := range values {
						if len(v) > 2 {
							ret[v] = id
						}
					}
				case string:
					if len(values) > 2 {
						ret[values] = id
					}
				}
			}
		}
	}
	w.WriteHeader(http.StatusOK)
	enc.Encode(ret)
	return nil
}

func (ws *WebServer) ClientHandler() *http.ServeMux {

	srv := http.NewServeMux()

	srv.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		defaultHeaders(w, r, false, "0")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	srv.HandleFunc("/content", JsonHandler(ws.Tracking, ws.ContentSearch))
	srv.HandleFunc("/facets", JsonHandler(ws.Tracking, ws.GetFacets))
	//srv.HandleFunc("/ai-search", JsonHandler(ws.Tracking, ws.SearchEmbeddings))
	srv.HandleFunc("/related/{id}", JsonHandler(ws.Tracking, ws.Related))
	srv.HandleFunc("/popular", JsonHandler(ws.Tracking, ws.Popular))
	srv.HandleFunc("/similar", JsonHandler(ws.Tracking, ws.Similar))
	srv.HandleFunc("/trigger-words", JsonHandler(ws.Tracking, ws.TriggerWords))
	srv.HandleFunc("/facet-list", JsonHandler(ws.Tracking, ws.Facets))
	srv.HandleFunc("/suggest", JsonHandler(ws.Tracking, ws.Suggest))
	//srv.HandleFunc("/categories", JsonHandler(ws.Tracking, ws.Categories))
	//srv.HandleFunc("/search", ws.QueryIndex)
	srv.HandleFunc("/stream", JsonHandler(ws.Tracking, ws.SearchStreamed))

	srv.HandleFunc("/ids", JsonHandler(ws.Tracking, ws.GetIds))
	srv.HandleFunc("GET /get/{id}", JsonHandler(ws.Tracking, ws.GetItem))
	srv.HandleFunc("POST /get", JsonHandler(ws.Tracking, ws.GetItems))
	srv.HandleFunc("/values/{id}", JsonHandler(ws.Tracking, ws.GetValues))

	return srv
}
