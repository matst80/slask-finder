package server

import (
	"encoding/json"
	"fmt"
	"iter"
	"log"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/matst80/slask-finder/pkg/facet"
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
		Help: "The total number facets",
	})
)

func (ws *WebServer) getStockResult(stockLocations []string) *types.ItemList {
	resultStockIds := &types.ItemList{}
	for _, stockId := range stockLocations {
		stockIds, ok := ws.Index.ItemsInStock[stockId]
		if ok {
			resultStockIds.Merge(&stockIds)
		}
	}
	return resultStockIds
}

func (ws *WebServer) getSearchAndStockResult(sr *types.FacetRequest) *types.ItemList {
	var initialIds *types.ItemList = nil
	//var documentResult *search.DocumentResult = nil
	if sr.Query != "" {
		if sr.Query == "*" {
			// probably should copy this
			clone := maps.Clone(ws.Index.All)
			initialIds = &clone

		} else {
			initialIds = ws.Index.Search.Search(sr.Query)

			//documentResult = queryResult
		}
	}

	if len(sr.Stock) > 0 {
		resultStockIds := ws.getStockResult(sr.Stock)

		if initialIds == nil {
			initialIds = resultStockIds
		} else {
			initialIds.Intersect(*resultStockIds)
		}
	}

	return initialIds
}

func (ws *WebServer) ContentSearch(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	query := r.URL.Query().Get("q")
	query = strings.TrimSpace(query)
	res := ws.ContentIndex.MatchQuery(query)
	defaultHeaders(w, r, false, "160")
	w.WriteHeader(http.StatusOK)
	var err error
	for content := range res {
		err = enc.Encode(content)
	}
	return err
}

func (ws *WebServer) GetFacets(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	s := time.Now()
	sr, err := GetFacetQueryFromRequest(r)
	if err != nil {
		return err
	}
	go facetSearches.Inc()
	baseIds := &types.ItemList{}
	ids := &types.ItemList{}

	qm := types.NewQueryMerger(ids)
	qm.Add(func() *types.ItemList {
		baseIds = ws.getSearchAndStockResult(sr)
		return baseIds
	})

	ws.Index.Match(sr.Filters, qm)

	ch := make(chan *index.JsonFacet)
	wg := &sync.WaitGroup{}

	qm.Wait()

	ws.getOtherFacets(ids, sr, ch, wg)
	ws.getSearchedFacets(baseIds, sr, ch, wg)

	// todo optimize
	go func() {
		wg.Wait()
		close(ch)
	}()

	ret := make([]*index.JsonFacet, 0)
	for item := range ch {
		if item != nil && (item.Result.HasValues() || item.Selected != nil) {
			ret = append(ret, item)
		}
	}

	publicHeaders(w, r, true, "600")
	w.Header().Set("x-duration", fmt.Sprintf("%v", time.Since(s)))
	w.WriteHeader(http.StatusOK)

	return enc.Encode(ws.Sorting.GetSortedFields(ret))
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
	s := time.Now()
	sr, err := GetQueryFromRequest(r)

	if err != nil {
		return err
	}
	log.Printf("search streamed %s", sr.Query)

	resultChan := make(chan searchResult)

	defer close(resultChan)

	go ws.getMatchAndSort(sr, resultChan)

	defaultHeaders(w, r, false, "10")
	w.WriteHeader(http.StatusOK)

	start := sr.PageSize * sr.Page
	end := start + sr.PageSize
	result := <-resultChan
	sortedItemsChan := make(chan iter.Seq[types.Item])
	go ws.Sorting.GetSortedItemsIterator(sessionId, result.sort, result.matching, start, sortedItemsChan, result.sortOverrides...)

	fn := <-sortedItemsChan
	idx := 0

	for item := range fn {
		idx++
		if idx <= start {
			continue
		}
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
	l := len(*result.matching)

	if ws.Tracking != nil && !sr.SkipTracking {
		go ws.Tracking.TrackSearch(sessionId, sr.Filters, l, sr.Query, sr.Page, r)
	}

	return enc.Encode(SearchResponse{
		Duration:  fmt.Sprintf("%v", time.Since(s)),
		Page:      sr.Page,
		PageSize:  sr.PageSize,
		Start:     start,
		End:       min(l, end),
		TotalHits: l,
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

	other := words[:len(words)-1]
	go noSuggests.Inc()

	wordMatchesChan := make(chan []search.Match)
	sortChan := make(chan *types.ByValue)
	defer close(wordMatchesChan)
	defer close(sortChan)

	docResult := ws.Index.Search.Search(query)
	types.Merge(results, *docResult)
	//results = *docResult

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
				//results.Merge(s.Items)
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

	sortedItemsChan := make(chan iter.Seq[types.Item])
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

	ws.getSuggestFacets(&results, &types.FacetRequest{Filters: &types.Filters{}}, ch, wg)

	_, err = w.Write([]byte("\n"))

	//ret := make(map[uint]*index.JsonFacet)
	go func() {
		wg.Wait()
		close(ch)
	}()
	for jsonFacet := range ch {
		if jsonFacet != nil && (jsonFacet.Result.HasValues() || jsonFacet.Selected != nil) {
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
	for j, v := range sortedItems {
		item, ok := ws.Index.Items[v.Id]
		if ok {
			err := enc.Encode(item)
			if err != nil {
				return err
			}
		}
		if j > 80 {
			break
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
	productTypeId := types.CurrentSettings.ProductTypeId
	wg := &sync.WaitGroup{}
	pop := ws.Sorting.GetSort("popular")
	delete(*fields, productTypeId)
	for id := range *items {
		if item, ok := ws.Index.Items[id]; ok {
			if itemType, typeOk := item.GetFields()[productTypeId]; typeOk {
				articleTypes[itemType.(string)]++
			}
		}
	}
	getSimilar := func(articleType string, ret chan *Similar, wg *sync.WaitGroup, sort *types.ByValue, popularity float64) {
		//ids := make(chan *types.ItemList)
		//defer close(ids)
		//defer wg.Done()
		filter := &types.Filters{
			StringFilter: []types.StringFilter{
				{Id: productTypeId, Value: []string{articleType}},
			},
		}
		resultIds := &types.ItemList{}
		qm := types.NewQueryMerger(resultIds)
		ws.Index.Match(filter, qm)
		qm.Wait()
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
				similar.Items = append(similar.Items, item)
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

type PossibleRelationQuery struct {
	Value interface{} `json:"value"`
	Id    uint        `json:"id"`
}

func (ws *WebServer) FindRelated(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	var query PossibleRelationQuery
	err := json.NewDecoder(r.Body).Decode(&query)
	if err != nil {
		return err
	}
	var base types.BaseField
	var l int
	res := make(map[uint]int)
	for _, f := range ws.Index.Facets {

		keyFacet, ok := f.(facet.KeyField)
		if !ok {
			continue
		}
		// if f.GetType() != types.FacetKeyType {
		// 	continue
		// }
		base = *keyFacet.BaseField
		if base.Id == query.Id || !base.Searchable {
			continue
		}
		keyValue, ok := types.AsKeyFilterValue(query.Value)
		if !ok {
			continue
		}
		matches := keyFacet.Match(keyValue)

		if matches != nil {
			l = len(*matches)
			if l > 0 {
				res[base.Id] = l
			}
		}
	}
	publicHeaders(w, r, true, "1200")
	if len(res) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}

	w.WriteHeader(http.StatusOK)
	return enc.Encode(res)
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
		if ok && item.GetId() != uint(id) {
			err = enc.Encode(item)
			i++
		}
		if i > 20 || err != nil {
			break
		}
	}
	return err
}

func (ws *WebServer) Compatible(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	excludedProductTypes := make([]string, 0)
	maxItems := 60
	limitString := r.URL.Query().Get("limit")
	if limitString != "" {
		limit, err := strconv.Atoi(limitString)
		if err != nil {
			maxItems = limit
		}
	}
	if r.Method != http.MethodGet {
		cartItemIds := make([]uint, 0)
		err := json.NewDecoder(r.Body).Decode(&cartItemIds)
		if err == nil {
			for _, id := range cartItemIds {
				if item, ok := ws.Index.Items[id]; ok {
					if productType, typeOk := item.GetFieldValue(types.CurrentSettings.ProductTypeId); typeOk {
						excludedProductTypes = append(excludedProductTypes, productType.(string))
					}
				}
			}
		}
		log.Printf("cart item ids %v", cartItemIds)
		log.Printf("excluded product types %v", excludedProductTypes)
	}
	idString := r.PathValue("id")
	id, err := strconv.Atoi(idString)
	if err != nil {
		return err
	}

	sortChan := make(chan *types.ByValue)
	defer close(sortChan)

	publicHeaders(w, r, false, "600")
	w.WriteHeader(http.StatusOK)

	go ws.Sorting.GetSorting("popular", sortChan)
	related, err := ws.Index.Compatible(uint(id))
	if err != nil {
		return err
	}
	i := 0

	ws.Index.Lock()
	defer ws.Index.Unlock()

	sort := <-sortChan
	for relatedId := range sort.SortMap(*related) {
		item, ok := ws.Index.Items[relatedId]
		if ok {
			if len(excludedProductTypes) > 0 {
				if productType, typeOk := item.GetFieldValue(types.CurrentSettings.ProductTypeId); typeOk {

					itemProductType := productType.(string)
					if slices.Contains(excludedProductTypes, itemProductType) {
						//log.Printf("skipping %d %s", item.GetId(), itemProductType)
						continue
					}

				}
			}

			err = enc.Encode(item)
			i++
		}
		if i > maxItems || err != nil {
			break
		}
	}
	return err
}

// type CategoryResult struct {
// 	Value    string            `json:"value"`
// 	Children []*CategoryResult `json:"children,omitempty"`
// }

// func CategoryResultFrom(c *index.Category) *CategoryResult {
// 	ret := &CategoryResult{}
// 	ret.Value = *c.Value
// 	ret.Children = make([]*CategoryResult, 0)
// 	if c.Children != nil {
// 		for _, child := range c.Children {
// 			if child != nil {
// 				ret.Children = append(ret.Children, CategoryResultFrom(child))
// 			}
// 		}
// 	}
// 	return ret
// }

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

func (ws *WebServer) GetItemBySku(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	sku := r.PathValue("sku")
	publicHeaders(w, r, true, "120")
	item, ok := ws.Index.ItemsBySku[sku]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}

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
//
//func (ws *WebServer) TriggerWords(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
//	defaultHeaders(w, r, true, "1200")
//	ret := make(map[string]uint)
//	for id, f := range ws.Index.Facets {
//		base := f.GetBaseField()
//		if (base.Type != "" || base.CategoryLevel > 0) && !base.HideFacet && f.GetType() == types.FacetKeyType {
//			for _, line := range f.GetValues() {
//				switch values := line.(type) {
//				case []string:
//					for _, v := range values {
//						if len(v) > 2 {
//							ret[v] = id
//						}
//					}
//				case string:
//					if len(values) > 2 {
//						ret[values] = id
//					}
//				}
//			}
//		}
//	}
//	w.WriteHeader(http.StatusOK)
//	enc.Encode(ret)
//	return nil
//}

func (ws *WebServer) ReloadSettings(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	defaultHeaders(w, r, true, "1200")
	w.WriteHeader(http.StatusOK)
	if err := ws.Db.LoadSettings(); err != nil {
		return err
	}

	return enc.Encode(types.CurrentSettings)
}

func (ws *WebServer) SearchEmbeddings(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	query := r.URL.Query().Get("q")
	query = strings.TrimSpace(query)

	if query == "" {
		defaultHeaders(w, r, true, "1200")
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("query parameter 'q' is required")
	}

	// Check if embeddings engine is available
	if ws.Index.EmbeddingsEngine == nil {
		return fmt.Errorf("embeddings engine not initialized")
	}
	start := time.Now()
	// Generate embeddings for the query
	queryEmbeddings, err := ws.Index.EmbeddingsEngine.GenerateEmbeddings(query)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}
	embeddingsDuration := time.Since(start)

	// Find similar items using cosine similarity
	matches := make(types.ItemList)
	// sortedItems := make(types.ByValue, 0)

	// similarityThreshold := 0.5 // Configurable threshold

	// Convert queryEmbeddings (float32) to float64 for cosine similarity calculation

	// Lock the index for read access
	ws.Index.Lock()
	defer ws.Index.Unlock()

	// Find items with similar embeddings
	start = time.Now()
	ids, _ := types.FindTopSimilarEmbeddings(queryEmbeddings, ws.Index.Embeddings, 60)
	// for itemID, itemEmb := range ws.Index.Embeddings {
	// 	// Convert item embeddings (float32) to float64 for cosine similarity calculation

	// 	similarity := types.CosineSimilarity(queryEmbeddings, itemEmb)

	// 	if similarity > similarityThreshold {
	// 		_, exists := ws.Index.Items[itemID]
	// 		if !exists {
	// 			continue
	// 		}

	// 		matches.AddId(itemID)
	// 		sortedItems = append(sortedItems, types.Lookup{
	// 			Id:    itemID,
	// 			Value: similarity,
	// 		})
	// 	}
	// }

	// // Sort by similarity (highest first)
	// slices.SortFunc(sortedItems, func(a, b types.Lookup) int {
	// 	return cmp.Compare(b.Value, a.Value)
	// })
	matchDuration := time.Since(start)
	defaultHeaders(w, r, true, "120")
	w.Header().Set("x-embeddings-duration", fmt.Sprintf("%v", embeddingsDuration))
	w.Header().Set("x-match-duration", fmt.Sprintf("%v", matchDuration))
	w.WriteHeader(http.StatusOK)

	// Prepare limit on results
	limit := 60
	limitParam := r.URL.Query().Get("limit")
	if limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = l
		}
	}

	// Stream the results to the client
	count := 0
	for _, matchId := range ids {
		if count >= limit {
			break
		}

		item, ok := ws.Index.Items[matchId]
		if ok {
			err := enc.Encode(item)
			if err != nil {
				return err
			}
			count++
		}
	}

	// Track search if tracking is enabled
	if ws.Tracking != nil {
		go ws.Tracking.TrackSearch(sessionId, nil, len(matches), query, 0, r)
	}

	return nil
}

func (ws *WebServer) CosineSimilar(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	idString := r.PathValue("id")
	id, err := strconv.Atoi(idString)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("invalid id: %s", idString)
	}
	iid := uint(id)
	item, ok := ws.Index.Embeddings[iid]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return fmt.Errorf("item not found with id: %d", id)
	}
	defaultHeaders(w, r, true, "120")
	w.WriteHeader(http.StatusOK)
	ids, _ := types.FindTopSimilarEmbeddings(item, ws.Index.Embeddings, 30)
	// Stream the results to the client

	for _, rid := range ids {
		if rid == iid {
			continue // Skip the item itself
		}
		item, ok := ws.Index.Items[rid]
		if ok {
			err := enc.Encode(item)
			if err != nil {
				return err
			}

		}
	}
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
	srv.HandleFunc("/compatible/{id}", JsonHandler(ws.Tracking, ws.Compatible))
	srv.HandleFunc("/popular", JsonHandler(ws.Tracking, ws.Popular))
	srv.HandleFunc("/natural", JsonHandler(ws.Tracking, ws.SearchEmbeddings))
	srv.HandleFunc("/similar", JsonHandler(ws.Tracking, ws.Similar))
	srv.HandleFunc("/cosine-similar/{id}", JsonHandler(ws.Tracking, ws.CosineSimilar))
	//srv.HandleFunc("/trigger-words", JsonHandler(ws.Tracking, ws.TriggerWords))
	srv.HandleFunc("/facet-list", JsonHandler(ws.Tracking, ws.Facets))
	srv.HandleFunc("/suggest", JsonHandler(ws.Tracking, ws.Suggest))
	srv.HandleFunc("/find-related", JsonHandler(ws.Tracking, ws.FindRelated))
	//srv.HandleFunc("/categories", JsonHandler(ws.Tracking, ws.Categories))
	//srv.HandleFunc("/search", ws.QueryIndex)
	srv.HandleFunc("GET /settings", ws.GetSettings)
	srv.HandleFunc("/stream", JsonHandler(ws.Tracking, ws.SearchStreamed))
	srv.HandleFunc("/reload-settings", JsonHandler(ws.Tracking, ws.ReloadSettings))
	srv.HandleFunc("GET /relation-groups", ws.HandleRelationGroups)

	srv.HandleFunc("/ids", JsonHandler(ws.Tracking, ws.GetIds))
	srv.HandleFunc("GET /get/{id}", JsonHandler(ws.Tracking, ws.GetItem))
	srv.HandleFunc("GET /by-sku/{sku}", JsonHandler(ws.Tracking, ws.GetItemBySku))
	srv.HandleFunc("POST /get", JsonHandler(ws.Tracking, ws.GetItems))
	srv.HandleFunc("/values/{id}", JsonHandler(ws.Tracking, ws.GetValues))

	return srv
}
