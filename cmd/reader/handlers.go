package main

import (
	"bufio"
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
	baseIds := &types.ItemList{}

	qm := types.NewQueryMerger(ids)

	ws.searchIndex.MatchQuery(sr.Query, qm)
	ws.itemIndex.MatchStock(sr.Stock, qm)
	qm.GetClone(baseIds)
	ws.facetHandler.Match(sr.Filters, qm)

	ch := make(chan *facet.JsonFacet)
	wg := &sync.WaitGroup{}

	qm.Wait()
	if baseIds.Len() == 0 {
		baseIds.Merge(ws.searchIndex.All)
	}
	ws.facetHandler.GetOtherFacets(ids, sr, ch, wg)
	ws.facetHandler.GetSearchedFacets(baseIds, sr, ch, wg)
	// todo optimize
	go func() {
		wg.Wait()
		close(ch)
	}()

	ret := make([]*facet.JsonFacet, 0)
	for jsonFacet := range ch {
		if jsonFacet == nil {
			continue
		}

		if jsonFacet.Result.HasValues() || jsonFacet.Selected != nil {
			ret = append(ret, jsonFacet)
		}
	}

	publicHeaders(w, r, true, "600")
	w.Header().Set("x-duration", fmt.Sprintf("%v", time.Since(s)))
	w.WriteHeader(http.StatusOK)
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

	sortedItemsItr := ws.sortingHandler.GetSortedItemsIterator(sessionId, sr.Sort, ids, start)

	idx := 0

	for item := range ws.itemIndex.GetItems(sortedItemsItr) {
		idx++

		_, err = item.Write(w)

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
	l := ids.Len()

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
	idStr := r.PathValue("id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return err
	}

	item, ok := ws.itemIndex.GetItem(types.ItemId(id64))
	if !ok {
		return fmt.Errorf("item %s not found", idStr)
	}
	publicHeaders(w, r, true, "120")
	w.WriteHeader(http.StatusOK)
	_, err = item.Write(w)
	return err
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
	_, err := item.Write(w)
	return err
}

func (ws *app) Related(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	idString := r.PathValue("id")
	id64, err := strconv.ParseUint(idString, 10, 64)
	if err != nil {
		return err
	}
	if id64 > uint64(^uint(0)) {
		return fmt.Errorf("item id out of range")
	}
	item, ok := ws.itemIndex.GetItem(types.ItemId(id64))
	if !ok {
		return fmt.Errorf("item %s not found", idString)
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

	for item := range ws.itemIndex.GetItems(ws.sortingHandler.GetSortedItemsIterator(sessionId, "popular", related, 0)) {
		if ok && item.GetId() != types.ItemId(id64) {
			_, err = item.Write(w)
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
		cartItemIds := make([]types.ItemId, 0)
		err := json.NewDecoder(r.Body).Decode(&cartItemIds)
		if err == nil {
			for item := range ws.itemIndex.GetItems(slices.Values(cartItemIds)) {

				if productType, typeOk := item.GetStringFieldValue(types.CurrentSettings.ProductTypeId); typeOk {
					excludedProductTypes = append(excludedProductTypes, productType)
				}

			}
		}
		// log.Printf("cart item ids %v", cartItemIds)
		// log.Printf("excluded product types %v", excludedProductTypes)
	}
	idString := r.PathValue("id")
	id64, err := strconv.ParseUint(idString, 10, 64)
	if err != nil {
		return err
	}
	if id64 > uint64(^uint(0)) {
		return fmt.Errorf("item id out of range")
	}
	item, ok := ws.itemIndex.GetItem(types.ItemId(id64))
	if !ok {
		return fmt.Errorf("item %s not found", idString)
	}

	publicHeaders(w, r, false, "600")
	w.WriteHeader(http.StatusOK)

	related, err := ws.facetHandler.Compatible(item)
	if err != nil {
		return err
	}
	i := 0

	for item := range ws.itemIndex.GetItems(ws.sortingHandler.GetSortedItemsIterator(sessionId, "popular", related, 0)) {

		if len(excludedProductTypes) > 0 {
			if productType, typeOk := item.GetStringFieldValue(types.CurrentSettings.ProductTypeId); typeOk {

				if slices.Contains(excludedProductTypes, productType) {
					//log.Printf("skipping %d %s", item.GetId(), itemProductType)
					continue
				}

			}
		}

		_, err = item.Write(w)
		i++

		if i > maxItems || err != nil {
			break
		}
	}
	return err
}

func (ws *app) GetValues(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	idString := r.PathValue("id")
	id64, err := strconv.ParseUint(idString, 10, 64)
	if err != nil {
		return err
	}
	if id64 > uint64(^uint(0)) {
		return fmt.Errorf("facet id out of range")
	}
	if field, ok := ws.facetHandler.GetFacet(types.FacetId(id64)); ok {
		w.WriteHeader(http.StatusOK)
		return enc.Encode(field.GetValues())
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
		if _, err := w.Write([]byte("\n")); err != nil { // write separator newline
			return err
		}
		max := 30
		for item := range ws.itemIndex.GetItems(ws.sortingHandler.GetSortedItemsIterator(sessionId, "popular", ws.searchIndex.All, 0)) {

			_, err := item.Write(w)
			if err != nil {
				return err
			}
			max--
			if max <= 0 {
				break
			}

		}
		if _, err := w.Write([]byte("\n")); err != nil { // final separator newline
			return err
		}
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

	(&results).Merge(docResult)

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
	hasResults := results.Len() > 0
	var err error
	for _, s := range <-wordMatchesChan {
		suggestResult.Prefix = lastWord
		suggestResult.Word = s.Word
		totalHits := s.Items.GetCardinality()
		if totalHits > 0 {
			if !hasResults {
				suggestResult.Hits = totalHits
				err = enc.Encode(suggestResult)
				//results.Merge(s.Items)
			} else {

				suggestResult.Hits = results.Bitmap().AndCardinality(s.Items)
				if suggestResult.Hits > 0 {
					err = enc.Encode(suggestResult)
				}
				// dont intersect with the other words yet since partial
				//results.Intersect(*s.Items)
			}
		}

	}
	if err != nil {
		return err
	}

	if _, err = w.Write([]byte("\n")); err != nil { // separator between suggestions and items
		return err
	}

	idx := 0
	for item := range ws.itemIndex.GetItems(ws.sortingHandler.GetSortedItemsIterator(sessionId, "popular", &results, 0)) {
		idx++
		_, err = item.Write(w)
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

	if _, err := w.Write([]byte("\n")); err != nil { // separator between items and facets
		return err
	}

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
	if _, err := w.Write([]byte("\n")); err != nil { // leading newline before items
		return err
	}
	max := 60
	for item := range ws.itemIndex.GetItems(ws.sortingHandler.GetSortedItemsIterator(sessionId, "popular", ws.searchIndex.All, 0)) {

		_, err := item.Write(w)
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
func (ws *app) GetFacetGroups(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	defaultHeaders(w, r, true, "1200")
	w.WriteHeader(http.StatusOK)
	return enc.Encode(types.CurrentSettings.FacetGroups)
}

func (ws *app) StreamItemsFromIds(w http.ResponseWriter, r *http.Request) {
	// Set appropriate headers for JSONL streaming
	w.Header().Set("Content-Type", "application/jsonl+json; charset=UTF-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	// Create scanner to read line by line
	scanner := bufio.NewScanner(r.Body)

	// Create iter.Seq[uint] to feed to GetItems
	idSeq := func(yield func(types.ItemId) bool) {
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue // Skip empty lines
			}

			// Convert string ID to uint
			id, err := strconv.ParseUint(line, 10, 32)
			if err != nil {
				log.Printf("Error parsing item ID '%s': %v", line, err)
				continue // Skip invalid IDs
			}

			if !yield(types.ItemId(id)) {
				return
			}
		}
	}

	// Stream items using GetItems
	for item := range ws.itemIndex.GetItems(idSeq) {

		if _, err := item.Write(w); err != nil {
			log.Printf("Error encoding item: %v", err)
			return
		}
	}

	// Check for scanning errors
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading request body: %v", err)
	}
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
