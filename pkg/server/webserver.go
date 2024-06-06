package server

import (
	"encoding/json"
	"log"
	"net/http"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
)

type WebServer struct {
	Index *index.Index
}

type SearchRequest struct {
	StringSearches []index.StringSearch `json:"string"`
	NumberSearches []index.NumberSearch `json:"number"`
}

type ValueResponse struct {
	Field  facet.Field `json:"field"`
	Values []string    `json:"values"`
}

type NumberValueResponse struct {
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
	Items  []index.Item  `json:"items"`
	Facets FacetResponse `json:"facets"`
}

func NewWebServer() WebServer {
	return WebServer{
		Index: index.NewIndex(),
	}
}

func toResponse(facets index.Facets) FacetResponse {
	fields := []ValueResponse{}
	for _, field := range facets.Fields {
		values := field.Values()
		if len(values) > 0 {
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
			numberFields = append(numberFields, NumberValueResponse{
				Field:  field.Field,
				Values: v.Values,
				Min:    v.Min,
				Max:    v.Max,
			})
		}
	}
	return FacetResponse{
		Fields:       fields,
		NumberFields: numberFields,
	}
}

func (ws *WebServer) Search(w http.ResponseWriter, r *http.Request) {
	var sr SearchRequest
	err := json.NewDecoder(r.Body).Decode(&sr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	matching := ws.Index.Match(sr.StringSearches, sr.NumberSearches)
	//w.WriteHeader(http.StatusOK)
	//w.Header().Set("content-type", "application/json; charset=utf-8")
	items := ws.Index.GetItems(matching)
	facets := ws.Index.GetFacetsFromResultIds(matching)
	data := SearchResponse{
		Items:  items,
		Facets: toResponse(facets),
	}

	encErr := json.NewEncoder(w).Encode(data)
	if encErr != nil {
		http.Error(w, encErr.Error(), http.StatusInternalServerError)
	}
}

func (ws *WebServer) StartServer() {

	http.HandleFunc("/search", ws.Search)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
