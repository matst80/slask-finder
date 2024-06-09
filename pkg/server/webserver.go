package server

import (
	"encoding/json"
	"log"
	"net/http"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/persistance"
	"tornberg.me/facet-search/pkg/search"
)

type WebServer struct {
	Index    *index.Index
	Db       *persistance.Persistance
	Sort     facet.SortIndex
	FreeText *search.FreeTextIndex
}

type NumberValueResponse struct {
	Field  facet.Field `json:"field"`
	Values []float64   `json:"values"`
	Min    float64     `json:"min"`
	Max    float64     `json:"max"`
}

type BoolValueResponse struct {
	Field  facet.Field `json:"field"`
	Values []bool      `json:"values"`
}

type SearchResponse struct {
	Items     []index.Item `json:"items"`
	Facets    index.Facets `json:"facets"`
	Page      int          `json:"page"`
	PageSize  int          `json:"pageSize"`
	TotalHits int          `json:"totalHits"`
}

func NewWebServer(db *persistance.Persistance) WebServer {
	return WebServer{
		Index: index.NewIndex(),
		Db:    db,
	}
}

type AddItemRequest []index.Item

func (ws *WebServer) Search(w http.ResponseWriter, r *http.Request) {

	sr, err := QueryFromRequest(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	itemsChan := make(chan []index.Item)
	facetsChan := make(chan index.Facets)

	matching := ws.Index.Match(sr.StringSearches, sr.NumberSearches, sr.BitSearches)
	//ids := matching.Ids()

	if !matching.HasItems() {
		w.WriteHeader(204)
		return
	}
	go func() {
		itemsChan <- ws.Index.GetItems(matching.SortedIds(ws.Sort, sr.PageSize*(sr.Page+1)), sr.Page, sr.PageSize)
	}()
	go func() {
		facetsChan <- ws.Index.GetFacetsFromResult(matching)
	}()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	data := SearchResponse{
		Items:     <-itemsChan,
		Facets:    <-facetsChan,
		Page:      sr.Page,
		PageSize:  sr.PageSize,
		TotalHits: matching.Length(),
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

	for _, item := range ws.Index.Items {
		ws.FreeText.AddDocument(search.Document{
			Id:     item.Id,
			Tokens: ws.FreeText.Tokenizer.Tokenize(item.Title),
		})

	}
	w.WriteHeader(http.StatusOK)
}

type SearchHit struct {
	Item  *index.Item `json:"item"`
	Score int         `json:"score"`
}

type SearchResult struct {
	Hits      []SearchHit `json:"items"`
	TotalHits int         `json:"totalHits"`
}

func (ws *WebServer) QueryIndex(w http.ResponseWriter, r *http.Request) {

	itemsChan := make(chan []SearchHit)
	query := ws.FreeText.Tokenizer.Tokenize(r.URL.Query().Get("q"))
	log.Printf("Query: %v", query)
	searchResults := ws.FreeText.Search(query)

	go func() {
		hits := make([]SearchHit, len(searchResults))
		idx := 0
		for id, score := range searchResults {
			item, ok := ws.Index.Items[id]
			if ok {
				hits[idx] = SearchHit{Item: &item, Score: score}
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

func (ws *WebServer) StartServer() {
	err := ws.Db.LoadIndex(ws.Index)
	priceSort := index.MakeSortFromNumberField(ws.Index.Items, 4)
	ws.Sort = priceSort
	ws.FreeText = search.NewFreeTextIndex(search.Tokenizer{MaxTokens: 128})
	if err != nil {
		log.Printf("Failed to load index %v", err)
	}
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	http.HandleFunc("/search", ws.Search)
	http.HandleFunc("/index", ws.IndexDocuments)
	http.HandleFunc("/query", ws.QueryIndex)
	http.HandleFunc("/add", ws.AddItem)
	http.HandleFunc("/save", ws.Save)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
