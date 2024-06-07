package server

import (
	"encoding/json"
	"log"
	"net/http"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/persistance"
)

type WebServer struct {
	Index *index.Index
	Db    *persistance.Persistance
}

type SearchRequest struct {
	StringSearches []index.StringSearch `json:"string"`
	NumberSearches []index.NumberSearch `json:"number"`
}

type ValueResponse struct {
	Id     int64       `json:"id"`
	Field  facet.Field `json:"field"`
	Values []string    `json:"values"`
}

type NumberValueResponse struct {
	Id     int64       `json:"id"`
	Field  facet.Field `json:"field"`
	Values []float64   `json:"values"`
	Min    float64     `json:"min"`
	Max    float64     `json:"max"`
}

type FacetResponse struct {
	Fields       []ValueResponse       `json:"fields"`
	NumberFields []NumberValueResponse `json:"numberFields"`
}

type SearchResponse struct {
	Items     []index.Item  `json:"items"`
	Facets    FacetResponse `json:"facets"`
	Page      int           `json:"page"`
	PageSize  int           `json:"pageSize"`
	TotalHits int           `json:"totalHits"`
}

func NewWebServer(db *persistance.Persistance) WebServer {
	return WebServer{
		Index: index.NewIndex(),
		Db:    db,
	}
}

type AddItemRequest []index.Item

func toResponse(facets index.Facets) FacetResponse {
	fields := []ValueResponse{}
	for _, field := range facets.Fields {
		values := field.Values()
		if len(values) > 0 {
			fields = append(fields, ValueResponse{
				Id:     field.Id,
				Field:  field.Field,
				Values: values,
			})
		}
	}
	numberFields := []NumberValueResponse{}
	for _, field := range facets.NumberFields {
		v := field.Values()
		if len(v.Values) > 0 {
			nr := NumberValueResponse{
				Id:    field.Id,
				Field: field.Field,

				Min: v.Min,
				Max: v.Max,
			}
			if len(v.Values) > 10 {
				// TODO Get median or something
			} else {
				nr.Values = v.Values
			}
			numberFields = append(numberFields, nr)

		}
	}
	return FacetResponse{
		Fields:       fields,
		NumberFields: numberFields,
	}
}

func (ws *WebServer) Search(w http.ResponseWriter, r *http.Request) {
	page := 0
	pageSize := 25
	var sr SearchRequest
	err := json.NewDecoder(r.Body).Decode(&sr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	matching := ws.Index.Match(sr.StringSearches, sr.NumberSearches)

	items := ws.Index.GetItems(matching, page, pageSize)
	if len(items) == 0 {
		w.WriteHeader(204)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	facets := ws.Index.GetFacetsFromResultIds(matching)
	data := SearchResponse{
		Items:     items,
		Facets:    toResponse(facets),
		Page:      page,
		PageSize:  pageSize,
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

func (ws *WebServer) StartServer() {
	err := ws.Db.LoadIndex(ws.Index)
	if err != nil {
		log.Printf("Failed to load index %v", err)
	}
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	http.HandleFunc("/search", ws.Search)
	http.HandleFunc("/add", ws.AddItem)
	http.HandleFunc("/save", ws.Save)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
