package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/types"
)

// func (ws *app) getSearchAndStockResult(sr *types.FacetRequest) *types.ItemList {
// 	var initialIds *types.ItemList = nil
// 	//var documentResult *search.DocumentResult = nil
// 	if sr.Query != "" {
// 		if sr.Query == "*" {
// 			// probably should copy this
// 			clone := maps.Clone(ws.facetHandler.All)
// 			initialIds = &clone

// 		} else {

// 			initialIds = ws.searchIndex.Search(sr.Query)

// 		}
// 	}
// 	if len(sr.Stock) > 0 {
// 		resultStockIds := ws.itemIndex.GetStockResult(sr.Stock)

// 		if initialIds == nil {
// 			initialIds = resultStockIds
// 		} else {
// 			initialIds.Intersect(*resultStockIds)
// 		}
// 	}

// 	return initialIds
// }

func (ws *app) GetFacets(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	s := time.Now()
	sr, err := types.GetFacetQueryFromRequest(r)
	if err != nil {
		return err
	}

	//baseIds := &types.ItemList{}
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

	//publicHeaders(w, r, true, "600")
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

	//defaultHeaders(w, r, false, "10")
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/jsonl; charset=UTF-8")

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

	// if ws.Tracking != nil && !sr.SkipTracking {
	// 	go ws.Tracking.TrackSearch(sessionId, sr.Filters, l, sr.Query, sr.Page, r)
	// }

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

func (a *app) UpdateSort(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	go a.sortingHandler.UpdateSorts()
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("ok"))
	return err
}

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
	//publicHeaders(w, r, true, "120")
	w.WriteHeader(http.StatusOK)
	return enc.Encode(item)
}

func (ws *app) GetItemBySku(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error {
	sku := r.PathValue("sku")
	//publicHeaders(w, r, true, "120")
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
	// sortChan := make(chan *types.ByValue)
	// defer close(sortChan)

	//publicHeaders(w, r, false, "600")
	w.WriteHeader(http.StatusOK)
	go func(ch chan *types.ItemList) {
		related, err := ws.facetHandler.Related(item)
		if err != nil {
			ch <- &types.ItemList{}
			return
		}
		ch <- related
	}(relatedChan)
	//go ws.Sorting.GetSorting("popular", sortChan)

	i := 0

	// ws.Index.Lock()
	// defer ws.Index.Unlock()
	related := <-relatedChan
	//sort := <-sortChan
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

	//publicHeaders(w, r, false, "600")
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
