package types

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/gorilla/schema"
)

type SearchRequest struct {
	*FacetRequest
	Filter       string `json:"filter" schema:"filter"`
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

	return decodeFiltersFromRequest(query, result.FacetRequest)
}

func GetFacetQueryFromRequest(r *http.Request) (*FacetRequest, error) {
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

func facetQueryFromRequestQuery(query url.Values, result *FacetRequest) error {

	err := decoder.Decode(result, query)
	if err != nil {
		return err
	}

	return decodeFiltersFromRequest(query, result)
}

func decodeFiltersFromRequest(query url.Values, result *FacetRequest) error {
	var err error
	key := map[FacetId]StringFilter{}
	rng := map[FacetId]RangeFilter{}
	for _, v := range query["str"] {
		parts := strings.Split(v, ":")
		if len(parts) != 2 {
			continue
		}
		idKey := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if idKey == "" || value == "" {
			continue
		}

		id64, err := strconv.ParseUint(idKey, 10, 64)
		id := FacetId(id64)
		if err != nil {
			continue
		}
		exclude := strings.HasPrefix(value, "!")
		if exclude {
			value = strings.TrimPrefix(value, "!")
		}

		if strings.Contains(value, "||") {
			key[id] = StringFilter{
				Id:    id,
				Not:   exclude,
				Value: strings.Split(value, "||"),
			}
		} else {

			key[id] = StringFilter{
				Id:    id,
				Not:   exclude,
				Value: []string{value},
			}

		}
	}

	for _, v := range query["rng"] {
		var id FacetId
		var _min float64
		var _max float64
		_, err := fmt.Sscanf(v, "%d:%f-%f", &id, &_min, &_max)

		if err != nil {
			continue
		}
		rng[id] = RangeFilter{
			Id:  id,
			Min: _min,
			Max: _max,
		}
	}
	result.RangeFilter = slices.Collect(maps.Values(rng))
	result.StringFilter = slices.Collect(maps.Values(key))
	result.Sanitize()
	return err
}

func makeBaseFacetRequest() *FacetRequest {
	return &FacetRequest{
		Filters: &Filters{
			StringFilter: []StringFilter{},
			RangeFilter:  []RangeFilter{},
		},
		IgnoreFacets: []FacetId{},
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
