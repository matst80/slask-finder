package main

import (
	"encoding/json"
	"fmt"
	"net/http"
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
