package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/schema"
	"github.com/matst80/slask-finder/pkg/facet"
	"github.com/matst80/slask-finder/pkg/index"
)

type FacetRequest struct {
	*index.Filters
	Stock []string `json:"stock" schema:"stock"`
	Query string   `json:"query" schema:"query"`
}

type SearchRequest struct {
	*FacetRequest
	Sort     string `json:"sort" schema:"sort,default:popular"`
	Page     int    `json:"page" schema:"page"`
	PageSize int    `json:"pageSize" schema:"size,default:40"`
}

func (s *SearchRequest) UseStaticPosition() bool {
	return s.Sort == "popular" || s.Sort == ""
}

func GetQueryFromRequest(r *http.Request, searchRequest *SearchRequest) error {
	if r.Method == http.MethodGet {
		return queryFromRequestQuery(r.URL.Query(), searchRequest)
	}
	return queryFromRequest(r, searchRequest)
}

func GetFacetQueryFromRequest(r *http.Request, facetRequest *FacetRequest) error {
	if r.Method == http.MethodGet {
		return facetQueryFromRequestQuery(r.URL.Query(), facetRequest)
	}
	return facetsQueryFromRequest(r, facetRequest)
}

func queryFromRequest(r *http.Request, searchRequest *SearchRequest) error {
	err := json.NewDecoder(r.Body).Decode(searchRequest)
	if err != nil {
		return err
	}
	return nil
}

func facetsQueryFromRequest(r *http.Request, searchRequest *FacetRequest) error {
	err := json.NewDecoder(r.Body).Decode(searchRequest)
	if err != nil {
		return err
	}
	return nil
}

func decodeFiltersFromRequest(query url.Values, result *FacetRequest) error {
	var err error
	for _, v := range query["str"] {
		parts := strings.Split(v, ":")
		if len(parts) != 2 {
			continue
		}
		id, err := strconv.Atoi(parts[0])

		if err != nil {
			continue
		}
		if strings.Contains(parts[1], "||") {
			result.StringFilter = append(result.StringFilter, facet.StringSearch{
				Id:    uint(id),
				Value: strings.Split(parts[1], "||"),
			})
		} else {
			result.StringFilter = append(result.StringFilter, facet.StringSearch{
				Id:    uint(id),
				Value: parts[1],
			})
		}
	}

	for _, v := range query["rng"] {
		var id uint
		var min float64
		var max float64
		_, err := fmt.Sscanf(v, "%d:%f-%f", &id, &min, &max)

		if err != nil {
			continue
		}
		result.RangeFilter = append(result.RangeFilter, facet.NumberSearch{
			Id:  uint(id),
			Min: min,
			Max: max,
		})
	}
	return err
}

func facetQueryFromRequestQuery(query url.Values, result *FacetRequest) error {
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	err := decoder.Decode(result, query)
	if err != nil {
		return err
	}

	decodeFiltersFromRequest(query, result)
	return err
}

func queryFromRequestQuery(query url.Values, result *SearchRequest) error {
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	err := decoder.Decode(result, query)
	if err != nil {
		return err
	}

	return decodeFiltersFromRequest(query, result.FacetRequest)
}
