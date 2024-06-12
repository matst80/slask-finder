package server

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/pprof"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/persistance"
	"tornberg.me/facet-search/pkg/search"
)

type WebServer struct {
	Index     *index.Index
	Db        *persistance.Persistance
	Sort      facet.SortIndex
	FieldSort facet.SortIndex
	FreeText  *search.FreeTextIndex
}

// type NumberValueResponse struct {
// 	Field  facet.NumberField[float64] `json:"field"`
// 	Values []float64   `json:"values"`
// 	Min    float64     `json:"min"`
// 	Max    float64     `json:"max"`
// }

// type BoolValueResponse struct {
// 	Field  facet.Field[bool] `json:"field"`
// 	Values []bool      `json:"values"`
// }

type SearchResponse struct {
	Items     []index.ResultItem `json:"items"`
	Facets    index.Facets       `json:"facets"`
	Page      int                `json:"page"`
	PageSize  int                `json:"pageSize"`
	TotalHits int                `json:"totalHits"`
}

func NewWebServer(db *persistance.Persistance) WebServer {
	return WebServer{
		Index: index.NewIndex(),
		Db:    db,
	}
}

type AddItemRequest []index.DataItem

func (ws *WebServer) Search(w http.ResponseWriter, r *http.Request) {

	sr, err := QueryFromRequest(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	itemsChan := make(chan []index.ResultItem)
	facetsChan := make(chan index.Facets)

	matching := ws.Index.Match(&sr.Filters)
	//ids := matching.Ids()

	// if !matching.HasItems() {
	// 	w.WriteHeader(204)
	// 	return
	// }
	go func() {
		itemsChan <- ws.Index.GetItems(matching.SortedIds(ws.Sort, sr.PageSize*(sr.Page+1)), sr.Page, sr.PageSize)
	}()
	go func() {
		facetsChan <- ws.Index.GetFacetsFromResult(&matching, &sr.Filters, &ws.FieldSort)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	data := SearchResponse{
		Items:     <-itemsChan,
		Facets:    <-facetsChan,
		Page:      sr.Page,
		PageSize:  sr.PageSize,
		TotalHits: len(matching),
	}

	encErr := json.NewEncoder(w).Encode(data)
	if encErr != nil {
		http.Error(w, encErr.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) AddItem(w http.ResponseWriter, r *http.Request) {
	items := AddItemRequest{}
	err := json.NewDecoder(r.Body).Decode(&items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, item := range items {
		ws.Index.AddItem(item)
	}
	w.WriteHeader(http.StatusOK)
}

func (ws *WebServer) Save(w http.ResponseWriter, r *http.Request) {
	err := ws.Db.SaveIndex(ws.Index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (ws *WebServer) IndexDocuments(w http.ResponseWriter, r *http.Request) {
	if ws.FreeText == nil {
		ws.FreeText = search.NewFreeTextIndex(search.Tokenizer{MaxTokens: 128})
	}
	for _, item := range ws.Index.Items {
		ws.FreeText.AddDocument(search.Document{
			Id:     item.Id,
			Tokens: ws.FreeText.Tokenizer.Tokenize(item.Title),
		})

	}
	ws.Db.SaveFreeText(ws.FreeText)
	w.WriteHeader(http.StatusOK)
}

// type SearchHit struct {
// 	Item  *index.Item `json:"item"`
// 	Score int         `json:"score"`
// }

type SearchResult struct {
	Hits      []*index.Item `json:"items"`
	TotalHits int           `json:"totalHits"`
}

func (ws *WebServer) QueryIndex(w http.ResponseWriter, r *http.Request) {

	itemsChan := make(chan []*index.Item)
	query := ws.FreeText.Tokenizer.Tokenize(r.URL.Query().Get("q"))
	log.Printf("Query: %v", query)
	searchResults := ws.FreeText.Search(query)

	go func() {
		hits := make([]*index.Item, len(searchResults))
		idx := 0

		res := searchResults.ToResultWithSort()
		ids := res.SortIndex.SortMap(res.IdList, 10000)
		for _, id := range ids {
			item, ok := ws.Index.Items[id]
			if ok {
				hits[idx] = &item
				idx++
			}
		}
		itemsChan <- hits[:idx]
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	result := SearchResult{Hits: <-itemsChan, TotalHits: len(searchResults)}
	err := json.NewEncoder(w).Encode(result)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) LoadDatabase() error {
	err := ws.Db.LoadIndex(ws.Index)
	if err != nil {
		//log.Printf("Failed to load index %v", err)
		return err
	}
	fieldSort := ws.Index.MakeSortForFields()
	priceSort := index.MakeSortFromDecimalField(ws.Index.Items, 4)
	ws.FieldSort = fieldSort
	ws.Sort = priceSort
	ws.FreeText = search.NewFreeTextIndex(search.Tokenizer{MaxTokens: 128})
	err = ws.Db.LoadFreeText(ws.FreeText)
	if err != nil {
		//log.Printf("Failed to load freetext %v", err)
		return err
	}
	return nil
}

func (ws *WebServer) StartServer() error {

	srv := http.NewServeMux()

	srv.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv.HandleFunc("/search", ws.Search)
	srv.HandleFunc("/index", ws.IndexDocuments)
	srv.HandleFunc("/query", ws.QueryIndex)
	srv.HandleFunc("/add", ws.AddItem)
	srv.HandleFunc("/save", ws.Save)

	srv.HandleFunc("/debug/pprof/", pprof.Index)
	srv.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	srv.HandleFunc("/debug/pprof/profile", pprof.Profile)
	srv.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	srv.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return http.ListenAndServe(":8080", srv)
}
