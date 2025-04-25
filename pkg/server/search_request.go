package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/schema"
	"github.com/matst80/slask-finder/pkg/types"
)

type SearchRequest struct {
	*types.FacetRequest
	SkipTracking bool   `json:"skipTracking" schema:"nt"`
	Sort         string `json:"sort" schema:"sort,default:popular"`
	Page         int    `json:"page" schema:"page"`
	PageSize     int    `json:"pageSize" schema:"size,default:40"`
}

var decoder = schema.NewDecoder()

func init() {
	decoder.IgnoreUnknownKeys(true)
}

func (s *SearchRequest) UseStaticPosition() bool {
	return s.Sort == "popular" || s.Sort == ""
}

func clamp[T int | float64](value, min, max T) T {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (s *SearchRequest) Sanitize() {
	s.Page = clamp(s.Page, 0, 100)
	s.PageSize = clamp(s.PageSize, 1, 1000)
	if s.Sort == "" {
		s.Sort = "popular"
	}
	s.FacetRequest.Sanitize()

}

func GetQueryFromRequest(r *http.Request) (*SearchRequest, error) {
	sr := makeBaseSearchRequest()
	var err error
	if r.Method == http.MethodGet {
		err = queryFromRequestQuery(r.URL.Query(), sr)
	} else {
		err = json.NewDecoder(r.Body).Decode(sr)
	}
	sr.Sanitize()
	return sr, err
}

func queryFromRequestQuery(query url.Values, result *SearchRequest) error {

	err := decoder.Decode(result, query)
	if err != nil {
		return err
	}
	result.Sanitize()
	return decodeFiltersFromRequest(query, result.FacetRequest)
}

func GetFacetQueryFromRequest(r *http.Request) (*types.FacetRequest, error) {
	sr := makeBaseFacetRequest()
	var err error
	if r.Method == http.MethodGet {
		err = facetQueryFromRequestQuery(r.URL.Query(), sr)
	} else {
		err = json.NewDecoder(r.Body).Decode(sr)
	}
	sr.Sanitize()
	return sr, err
}

func facetQueryFromRequestQuery(query url.Values, result *types.FacetRequest) error {

	err := decoder.Decode(result, query)
	if err != nil {
		return err
	}

	return decodeFiltersFromRequest(query, result)
}

func decodeFiltersFromRequest(query url.Values, result *types.FacetRequest) error {
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
			result.StringFilter = append(result.StringFilter, types.StringFilter{
				Id:    uint(id),
				Value: strings.Split(parts[1], "||"),
			})
		} else {
			result.StringFilter = append(result.StringFilter, types.StringFilter{
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
		result.RangeFilter = append(result.RangeFilter, types.RangeFilter{
			Id:  id,
			Min: _min,
			Max: _max,
		})
	}
	result.Sanitize()
	return err
}

func makeBaseFacetRequest() *types.FacetRequest {
	return &types.FacetRequest{
		Filters: &types.Filters{
			StringFilter: []types.StringFilter{},
			RangeFilter:  []types.RangeFilter{},
		},
		IgnoreFacets: []uint{},
		Stock:        []string{},
		Query:        "",
	}
}

func makeBaseSearchRequest() *SearchRequest {
	return &SearchRequest{
		FacetRequest: makeBaseFacetRequest(),
		Sort:         "popular",
		Page:         0,
		PageSize:     40,
	}
}
