package server

import (
	"encoding/json"
	"log"
	"maps"
	"net/http"
	"sync"

	"github.com/matst80/slask-finder/pkg/common"
	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/tracking"

	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/types"
)

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

func (ws *ClientWebServer) getMatchAndSort(sr *SearchRequest, result chan<- searchResult) {
	ids := &types.ItemList{}
	sortChan := make(chan *types.ByValue)
	go noSearches.Inc()

	defer close(sortChan)
	qm := types.NewQueryMerger(ids)
	qm.Add(func() *types.ItemList {
		return ws.getSearchAndStockResult(sr.FacetRequest)
	})

	ws.FacetHandler.Match(sr.Filters, qm)

	go ws.Sorting.GetSorting(sr.Sort, sortChan)

	qm.Wait()
	if sr.Filter != "" {
		if ws.SearchHandler != nil {
			ws.SearchHandler.Filter(sr.Filter, ids)
		}
	}
	result <- searchResult{
		matching:      ids,
		sort:          <-sortChan,
		sortOverrides: []index.SortOverride{},
	}
}

//
//const (
//	BucketSections = 20
//)
//
//type ResultBucket struct {
//	bucket [BucketSections]uint
//}
//func (r *ResultBucket) AddValue(value uint) {
//}

type FacetResultHandler struct {
	wg       *sync.WaitGroup
	c        chan *index.JsonFacet
	ids      *types.ItemList
	modifier func(*index.JsonFacet) *index.JsonFacet
}

func NewFacetResultHandler(modifyResult func(*index.JsonFacet) *index.JsonFacet) *FacetResultHandler {
	return &FacetResultHandler{
		wg:       &sync.WaitGroup{},
		c:        make(chan *index.JsonFacet),
		modifier: modifyResult,
	}
}

func (fh *FacetResultHandler) HandleKeyField(f facet.KeyField, ids *types.ItemList, selected interface{}) {
	defer fh.wg.Done()
	hasValues := false
	r := make(map[string]int, len(f.Keys))
	count := 0

	for key, sourceIds := range f.Keys {

		count = sourceIds.IntersectionLen(*ids)

		if count > 0 {
			hasValues = true
			r[key] = count
		}
	}
	if !hasValues {
		fh.c <- nil
		return
	}
	fh.c <- fh.modifier(&index.JsonFacet{
		BaseField: f.BaseField,
		Selected:  selected,
		Result:    &index.KeyFieldResult{Values: r},
	})
}

func (fh *FacetResultHandler) HandleIntegerField(f facet.IntegerField, ids *types.ItemList, selected interface{}) {
	defer fh.wg.Done()
	r := f.GetExtents(ids)
	if r == nil {
		fh.c <- nil
		return
	}
	fh.c <- fh.modifier(&index.JsonFacet{
		BaseField: f.BaseField,
		Selected:  selected,
		Result:    r,
	})
}

func (fh *FacetResultHandler) HandleDecimalField(f facet.DecimalField, ids *types.ItemList, selected interface{}) {
	defer fh.wg.Done()
	r := f.GetExtents(ids)
	if r == nil {
		fh.c <- nil
		return
	}
	fh.c <- fh.modifier(&index.JsonFacet{
		BaseField: f.BaseField,
		Selected:  selected,
		Result:    r,
	})
}

func (fh *FacetResultHandler) Handle(f types.Facet, selected interface{}) {
	if f.IsExcludedFromFacets() {
		return
	}
	switch field := f.(type) {
	case facet.KeyField:
		fh.wg.Add(1)
		go fh.HandleKeyField(field, fh.ids, selected)
	case facet.DecimalField:
		fh.wg.Add(1)
		go fh.HandleDecimalField(field, fh.ids, selected)
	case facet.IntegerField:
		fh.wg.Add(1)
		go fh.HandleIntegerField(field, fh.ids, selected)
	}
}

func getFacetResult(f types.Facet, baseIds *types.ItemList, c chan *index.JsonFacet, wg *sync.WaitGroup, selected interface{}) {
	defer wg.Done()

	baseField := f.GetBaseField()
	if baseField.HideFacet {
		c <- nil
		return
	}

	switch field := f.(type) {
	case facet.KeyField:
		hasValues := false
		r := make(map[string]int, len(field.Keys))
		count := 0
		//var ok bool
		for key, sourceIds := range field.Keys {
			count = sourceIds.IntersectionLen(*baseIds)

			if count > 0 {
				hasValues = true
				r[key] = count
			}
		}
		if !hasValues {
			c <- nil
			return
		}
		c <- &index.JsonFacet{
			BaseField: baseField,
			Selected:  selected,
			Result: &index.KeyFieldResult{
				Values: r,
			},
		}

	case facet.IntegerField:

		r := field.GetExtents(baseIds)
		if r == nil {
			c <- nil
			return
		}
		c <- &index.JsonFacet{
			BaseField: baseField,
			Selected:  selected,
			Result:    r,
		}
	case facet.DecimalField:
		r := field.GetExtents(baseIds)
		if r == nil {
			c <- nil
			return
		}
		c <- &index.JsonFacet{
			BaseField: baseField,
			Selected:  selected,
			Result:    r,
		}

	}
}

func (ws *ClientWebServer) getSearchedFacets(baseIds *types.ItemList, sr *types.FacetRequest, ch chan *index.JsonFacet, wg *sync.WaitGroup) {

	makeQm := func(list *types.ItemList) *types.QueryMerger {
		qm := types.NewQueryMerger(list)
		if baseIds != nil {
			qm.Add(func() *types.ItemList {
				return baseIds
			})
		}
		return qm
	}
	for _, s := range sr.StringFilter {
		if sr.IsIgnored(s.Id) {
			continue
		}
		var f types.Facet
		var faceExists bool
		if ws.FacetHandler != nil {
			f, faceExists = ws.FacetHandler.Facets[s.Id]
		}
		if faceExists && !f.IsExcludedFromFacets() {

			wg.Add(1)

			go func(otherFilters *types.Filters) {
				matchIds := &types.ItemList{}
				qm := makeQm(matchIds)
				ws.FacetHandler.Match(otherFilters, qm)
				qm.Wait()

				getFacetResult(f, matchIds, ch, wg, s.Value)
			}(sr.WithOut(s.Id, f.IsCategory()))

		}
	}
	for _, r := range sr.RangeFilter {
		var f types.Facet
		var facetExists bool
		if ws.FacetHandler != nil {
			f, facetExists = ws.FacetHandler.Facets[r.Id]
		}
		if facetExists && !sr.IsIgnored(r.Id) {
			wg.Add(1)
			go func(otherFilters *types.Filters) {
				matchIds := &types.ItemList{}
				qm := makeQm(matchIds)
				ws.FacetHandler.Match(otherFilters, qm)
				qm.Wait()
				go getFacetResult(f, matchIds, ch, wg, r)
			}(sr.WithOut(r.Id, false))
		}
	}
}

func (ws *ClientWebServer) getSuggestFacets(baseIds *types.ItemList, sr *types.FacetRequest, ch chan *index.JsonFacet, wg *sync.WaitGroup) {
	for _, id := range types.CurrentSettings.SuggestFacets {
		var f types.Facet
		var facetExists bool
		if ws.FacetHandler != nil {
			f, facetExists = ws.FacetHandler.Facets[id]
		}
		if facetExists && !f.IsExcludedFromFacets() {
			wg.Add(1)
			go getFacetResult(f, baseIds, ch, wg, nil)
		}
	}
}

func (ws *ClientWebServer) getOtherFacets(baseIds *types.ItemList, sr *types.FacetRequest, ch chan *index.JsonFacet, wg *sync.WaitGroup) {

	fieldIds := make(map[uint]struct{})
	limit := 30
	resultCount := len(*baseIds)
	t := 0
	for id := range *baseIds {
		var itemFieldIds types.ItemList
		var ok bool
		if ws.FacetHandler != nil {
			itemFieldIds, ok = ws.FacetHandler.ItemFieldIds[id]
		}
		if ok {
			maps.Copy(fieldIds, itemFieldIds)
			t++
		}
		if t > 1500 {
			break
		}
	}

	count := 0
	//var base *types.BaseField = nil
	if resultCount == 0 {
		var mainCat types.Facet
		if ws.FacetHandler != nil {
			mainCat = ws.FacetHandler.Facets[10] // todo setting
		}
		if mainCat != nil {
			//base = mainCat.GetBaseField()
			wg.Add(1)
			go getFacetResult(mainCat, &ws.Index.All, ch, wg, nil)
		}
	} else {

		for id := range ws.Sorting.FieldSorting.GetFieldSort().SortMap(fieldIds) {
			if count > limit {
				break
			}

			if !sr.Filters.HasField(id) && !sr.IsIgnored(id) {
				var f types.Facet
				var facetExists bool
				if ws.FacetHandler != nil {
					f, facetExists = ws.FacetHandler.Facets[id]
				}
				if facetExists && !f.IsExcludedFromFacets() {

					wg.Add(1)
					go getFacetResult(f, baseIds, ch, wg, nil)

					count++

				}
			} else {
				// log.Printf("Facet %d is in filters", id)
			}
		}
	}
}

func JsonHandler(trk tracking.Tracking, fn func(w http.ResponseWriter, r *http.Request, sessionId int, enc *json.Encoder) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			RespondToOptions(w, r)
			return
		}
		sessionId := common.HandleSessionCookie(trk, w, r)

		err := fn(w, r, sessionId, json.NewEncoder(w))
		if err != nil {
			log.Printf("Error handling request: %v", err)
		}
	}
}

func RespondToOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=3600")
	origin := r.Header.Get("Origin")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	w.Header().Set("Age", "0")
	w.WriteHeader(http.StatusAccepted)
}

func (ws *AdminWebServer) SaveHandleRelationGroups(w http.ResponseWriter, r *http.Request) {
	//if r.Method == "POST" {
	types.CurrentSettings.Lock()
	err := json.NewDecoder(r.Body).Decode(&types.CurrentSettings.FacetRelations)
	types.CurrentSettings.Unlock()

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = ws.Db.SaveSettings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// }
	// defaultHeaders(w, r, true, "1200")
	// w.WriteHeader(http.StatusOK)
	// err := json.NewEncoder(w).Encode(types.CurrentSettings.FacetRelations)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// }
}

func (ws *ClientWebServer) GetRelationGroups(w http.ResponseWriter, r *http.Request) {
	defaultHeaders(w, r, true, "1200")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(types.CurrentSettings.FacetRelations)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
