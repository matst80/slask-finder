package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/search"
	"github.com/matst80/slask-finder/pkg/types"
)

func (ws *app) GetFacets(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	s := time.Now()
	sr, err := types.GetFacetQueryFromRequest(r)
	if err != nil {
		return err
	}

	ids := &types.ItemList{}

	qm := types.NewQueryMerger(ids)

	ws.searchIndex.MatchQuery(sr.Query, qm)
	ws.itemIndex.MatchStock(sr.Stock, qm)
	ws.facetHandler.Match(sr.Filters, qm)

	ch := make(chan *facet.JsonFacet)
	wg := &sync.WaitGroup{}

	qm.Wait()

	ws.facetHandler.GetOtherFacets(ids, sr, ch, wg)
	ws.facetHandler.GetSearchedFacets(ids, sr, ch, wg)

	// todo optimize
	go func() {
		wg.Wait()
		close(ch)
	}()

	ret := make([]*facet.JsonFacet, 0)
	for item := range ch {
		if item != nil && (item.Result.HasValues() || item.Selected != nil) {
			ret = append(ret, item)
		}
	}

	w.WriteHeader(http.StatusOK)
	publicHeaders(w, r, true, "600")
	w.Header().Set("x-duration", fmt.Sprintf("%v", time.Since(s)))
	ws.facetHandler.SortJsonFacets(ret)
	return enc.Encode(ret)
}

func (ws *app) SearchStreamed(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	s := time.Now()
	sr, err := types.GetQueryFromRequest(r)

	if err != nil {
		return err
	}
	ids := &types.ItemList{}
	qm := types.NewQueryMerger(ids)
	ws.searchIndex.MatchQuery(sr.Query, qm)
	ws.itemIndex.MatchStock(sr.Stock, qm)

	ws.facetHandler.Match(sr.Filters, qm)

	w.WriteHeader(http.StatusOK)
	defaultHeaders(w, r, false, "10")

	start := sr.PageSize * sr.Page
	end := start + sr.PageSize

	qm.Wait()
	fn := ws.sortingHandler.GetSortedItemsIterator(sessionId, sr.Sort, *ids, start)

	idx := 0

	for item := range ws.itemIndex.GetItems(fn) {

		err = enc.Encode(item)
		idx++

		if idx >= sr.PageSize {
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
	l := len(*ids)

	if ws.tracker != nil && !sr.SkipTracking {
		go ws.tracker.TrackSearch(sessionId, sr.Filters, l, sr.Query, sr.Page, r)
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

// func (a *app) UpdateSort(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
// 	go a.sortingHandler.UpdateSorts()
// 	w.WriteHeader(http.StatusOK)
// 	_, err := w.Write([]byte("ok"))
// 	return err
// }

func (ws *app) GetItem(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	id := r.PathValue("id")
	itemId, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	item, ok := ws.itemIndex.GetItem(uint(itemId))
	if !ok {
		return err
	}
	publicHeaders(w, r, true, "120")
	w.WriteHeader(http.StatusOK)
	return enc.Encode(item)
}

func (ws *app) GetItemBySku(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	sku := r.PathValue("sku")
	publicHeaders(w, r, true, "120")
	item, ok := ws.itemIndex.GetItemBySku(sku)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}

	w.WriteHeader(http.StatusOK)
	return enc.Encode(item)
}

func (ws *app) Related(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {

	idString := r.PathValue("id")
	id, err := strconv.Atoi(idString)
	if err != nil {
		return err
	}

	item, ok := ws.itemIndex.GetItem(uint(id))
	if !ok {
		return fmt.Errorf("item %d not found", id)
	}

	relatedChan := make(chan *types.ItemList)
	defer close(relatedChan)

	publicHeaders(w, r, false, "600")
	w.WriteHeader(http.StatusOK)
	go func(ch chan *types.ItemList) {
		related, err := ws.facetHandler.Related(item)
		if err != nil {
			ch <- &types.ItemList{}
			return
		}
		ch <- related
	}(relatedChan)

	i := 0
	related := <-relatedChan

	for item := range ws.itemIndex.GetItems(ws.sortingHandler.GetSortedItemsIterator(sessionId, "popular", *related, 0)) {

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

func (ws *app) Compatible(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
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
			for item := range ws.itemIndex.GetItems(slices.Values(cartItemIds)) {

				if productType, typeOk := item.GetFieldValue(types.CurrentSettings.ProductTypeId); typeOk {
					excludedProductTypes = append(excludedProductTypes, productType.(string))
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
	item, ok := ws.itemIndex.GetItem(uint(id))
	if !ok {
		return fmt.Errorf("item %d not found", id)
	}

	publicHeaders(w, r, false, "600")
	w.WriteHeader(http.StatusOK)

	related, err := ws.facetHandler.Compatible(item)
	if err != nil {
		return err
	}
	i := 0

	for item := range ws.itemIndex.GetItems(ws.sortingHandler.GetSortedItemsIterator(sessionId, "popular", *related, 0)) {

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

		if i > maxItems || err != nil {
			break
		}
	}
	return err
}

func (ws *app) GetValues(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	idString := r.PathValue("id")
	id, err := strconv.Atoi(idString)
	if err != nil {
		return err
	}

	if field, ok := ws.facetHandler.GetFacet(uint(id)); ok {
		w.WriteHeader(http.StatusOK)
		err := enc.Encode(field.GetValues())
		return err
	}

	w.WriteHeader(http.StatusNotFound)
	return nil
}

func (ws *app) Facets(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	//publicHeaders(w, r, true, "1200")

	w.WriteHeader(http.StatusOK)

	return enc.Encode(slices.Collect(ws.facetHandler.GetAll()))
}

func (ws *app) Suggest(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {

	query := r.URL.Query().Get("q")
	if query == "" {

		defaultHeaders(w, r, true, "60")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("\n"))
		max := 30
		for item := range ws.itemIndex.GetItems(ws.sortingHandler.GetSortedItemsIterator(sessionId, "popular", ws.searchIndex.All, 0)) {

			err := enc.Encode(item)
			if err != nil {
				return err
			}
			max--
			if max <= 0 {
				break
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
	//go noSuggests.Inc()

	wordMatchesChan := make(chan []search.Match)
	sortChan := make(chan *types.ByValue)
	defer close(wordMatchesChan)
	defer close(sortChan)

	docResult := ws.searchIndex.Search(query)

	types.Merge(results, *docResult)

	// Use previous word to rank suggestions via Markov chain if available
	prevWord := ""
	if len(other) > 0 {
		prevWord = other[len(other)-1]
	}

	go ws.searchIndex.FindTrieMatchesForContext(prevWord, lastWord, wordMatchesChan)

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

	w.Write([]byte("\n"))

	idx := 0
	for item := range ws.itemIndex.GetItems(ws.sortingHandler.GetSortedItemsIterator(sessionId, "popular", results, 0)) {
		idx++
		err = enc.Encode(item)
		if idx >= 20 || err != nil {
			break
		}
	}
	if err != nil {
		return err
	}

	ch := make(chan *facet.JsonFacet)
	wg := &sync.WaitGroup{}

	ws.facetHandler.GetSuggestFacets(&results, &types.FacetRequest{Filters: &types.Filters{}}, ch, wg)

	w.Write([]byte("\n"))

	go func() {
		wg.Wait()
		close(ch)
	}()
	for jsonFacet := range ch {
		if jsonFacet != nil && (jsonFacet.Result.HasValues() || jsonFacet.Selected != nil) {
			err = enc.Encode(jsonFacet)
		}
	}
	return err
}

func (ws *app) Popular(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	//items, _ := ws.Sorting.GetSessionData(uint(sessionId))
	defaultHeaders(w, r, true, "60")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("\n"))
	max := 60
	for item := range ws.itemIndex.GetItems(ws.sortingHandler.GetSortedItemsIterator(sessionId, "popular", ws.searchIndex.All, 0)) {

		err := enc.Encode(item)
		if err != nil {
			return err
		}
		max--
		if max <= 0 {
			break
		}

	}
	return nil
}

func (ws *app) GetRelationGroups(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	defaultHeaders(w, r, true, "1200")
	w.WriteHeader(http.StatusOK)
	return enc.Encode(types.CurrentSettings.FacetRelations)
}

func (ws *app) SaveTrigger(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	log.Printf("Got save trigger")
	ws.gotSaveTrigger = true
	w.WriteHeader(http.StatusOK)
	return nil
}

// type Similar struct {
// 	ProductType string       `json:"productType"`
// 	Count       int          `json:"count"`
// 	Popularity  float64      `json:"popularity"`
// 	Items       []types.Item `json:"items"`
// }

// func (ws *app) Similar(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
// 	//items, fields := ws.Sorting.GetSessionData(uint(sessionId))
// 	articleTypes := map[string]float64{}
// 	itemChan := make(chan *Similar)
// 	productTypeId := types.CurrentSettings.ProductTypeId
// 	wg := &sync.WaitGroup{}
// 	//pop := ws.Sorting.GetSort("popular")
// 	//delete(*fields, productTypeId)
// 	for item := range ws.itemIndex.GetItems(ws.sortingHandler.GetSortedItemsIterator(sessionId, "popular", ws.searchIndex.All, 0)) {

// 		if itemType, typeOk := item.GetFields()[productTypeId]; typeOk {
// 			articleTypes[itemType.(string)]++
// 		}

// 	}
// 	getSimilar := func(articleType string, ret chan *Similar, wg *sync.WaitGroup, popularity float64) {
// 		//ids := make(chan *types.ItemList)
// 		//defer close(ids)
// 		//defer wg.Done()
// 		filter := &types.Filters{
// 			StringFilter: []types.StringFilter{
// 				{Id: productTypeId, Value: []string{articleType}},
// 			},
// 		}
// 		resultIds := &types.ItemList{}
// 		qm := types.NewQueryMerger(resultIds)
// 		ws.facetHandler.Match(filter, qm)
// 		qm.Wait()
// 		l := len(*resultIds)
// 		limit := min(l, 40)
// 		similar := Similar{
// 			ProductType: articleType,
// 			Count:       l,
// 			Popularity:  popularity,
// 			Items:       make([]types.Item, 0, limit),
// 		}
// 		for item := range ws.itemIndex.GetItems(ws.sortingHandler.GetSortedItemsIterator(sessionId, "popular", ws.searchIndex.All, 0)) {
// 			// if _, found := (*items)[id]; found {
// 			// 	continue
// 			// }
// 			// if item, ok := ws.Index.Items[id]; ok {
// 			similar.Items = append(similar.Items, item)
// 			if len(similar.Items) >= limit {
// 				break
// 			}
// 			//}
// 		}
// 		ret <- &similar
// 	}

// 	for i, typeValue := range sorting.ToSortedMap(articleTypes) {
// 		if i > 4 {
// 			break
// 		}
// 		wg.Add(1)
// 		go getSimilar(typeValue, itemChan, wg, articleTypes[typeValue])
// 	}
// 	go func() {
// 		wg.Wait()
// 		close(itemChan)
// 	}()
// 	defaultHeaders(w, r, false, "600")
// 	w.WriteHeader(http.StatusOK)
// 	for similar := range itemChan {
// 		err := enc.Encode(similar)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

func defaultHeaders(w http.ResponseWriter, r *http.Request, isJson bool, cacheTime string) {

	w.Header().Set("Cache-Control", "private, stale-while-revalidate="+cacheTime)
	genericHeaders(w, r, isJson)
}

func genericHeaders(w http.ResponseWriter, r *http.Request, isJson bool) {
	if isJson {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	} else {
		w.Header().Set("Content-Type", "application/jsonl+json; charset=UTF-8")
	}
	origin := r.Header.Get("Origin")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	w.Header().Set("Age", "0")
}

func publicHeaders(w http.ResponseWriter, r *http.Request, isJson bool, cacheTime string) {
	w.Header().Set("Cache-Control", "public, max-age="+cacheTime)
	genericHeaders(w, r, isJson)
}
