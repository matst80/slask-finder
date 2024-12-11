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

var decoder = schema.NewDecoder()

func init() {
	decoder.IgnoreUnknownKeys(true)
}

func (s *SearchRequest) UseStaticPosition() bool {
	return s.Sort == "popular" || s.Sort == ""
}

func GetQueryFromRequest(r *http.Request, searchRequest *SearchRequest) error {
	if r.Method == http.MethodGet {
		return queryFromRequestQuery(r.URL.Query(), searchRequest)
	}
	return json.NewDecoder(r.Body).Decode(searchRequest)
}

func queryFromRequestQuery(query url.Values, result *SearchRequest) error {

	err := decoder.Decode(result, query)
	if err != nil {
		return err
	}

	return decodeFiltersFromRequest(query, result.FacetRequest)
}

func GetFacetQueryFromRequest(r *http.Request, facetRequest *FacetRequest) error {
	if r.Method == http.MethodGet {
		return facetQueryFromRequestQuery(r.URL.Query(), facetRequest)
	}
	return json.NewDecoder(r.Body).Decode(facetRequest)
}

func facetQueryFromRequestQuery(query url.Values, result *FacetRequest) error {

	err := decoder.Decode(result, query)
	if err != nil {
		return err
	}

	return decodeFiltersFromRequest(query, result)
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
			result.StringFilter = append(result.StringFilter, facet.StringFilter{
				Id:    uint(id),
				Value: strings.Split(parts[1], "||"),
			})
		} else {
			result.StringFilter = append(result.StringFilter, facet.StringFilter{
				Id:    uint(id),
				Value: parts[1],
			})
		}
	}

	for _, v := range query["rng"] {
		var id uint
		var _min float64
		var _max float64
		_, err := fmt.Sscanf(v, "%d:%f-%f", &id, &_min, &_max)

		if err != nil {
			continue
		}
		result.RangeFilter = append(result.RangeFilter, facet.RangeFilter{
			Id:  id,
			Min: _min,
			Max: _max,
		})
	}
	return err
}
