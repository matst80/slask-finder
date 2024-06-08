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

type ValueResponse struct {
	facet.Field `json:"field"`
	Values      []string `json:"values"`
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

type FacetResponse struct {
	Fields       []ValueResponse       `json:"fields"`
	NumberFields []NumberValueResponse `json:"numberFields"`
	BoolFields   []BoolValueResponse   `json:"boolFields"`
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
		if len(values) > 1 {
			fields = append(fields, ValueResponse{
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
	boolFields := []BoolValueResponse{}
	for _, field := range facets.BoolFields {
		values := field.Values()
		if len(values) > 0 {
			boolFields = append(boolFields, BoolValueResponse{
				Field:  field.Field,
				Values: values,
			})
		}
	}
	return FacetResponse{
		Fields:       fields,
		NumberFields: numberFields,
		BoolFields:   boolFields,
	}
}

func (ws *WebServer) Search(w http.ResponseWriter, r *http.Request) {

	sr, err := QueryFromRequest(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	itemsChan := make(chan []index.Item)
	facetsChan := make(chan index.Facets)

	matching := ws.Index.Match(sr.StringSearches, sr.NumberSearches, sr.BitSearches)
	ids := matching.Ids()

	if len(ids) == 0 {
		w.WriteHeader(204)
		return
	}
	go func() {
		itemsChan <- ws.Index.GetItems(ids, sr.Page, sr.PageSize)
	}()
	go func() {
		facetsChan <- ws.Index.GetFacetsFromResult(matching)
	}()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	data := SearchResponse{
		Items:     <-itemsChan,
		Facets:    toResponse(<-facetsChan),
		Page:      sr.Page,
		PageSize:  sr.PageSize,
		TotalHits: len(ids),
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
