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

type SearchRequest struct {
	*index.Filters
	Stock    []string `json:"stock" schema:"stock"`
	Query    string   `json:"query" schema:"query"`
	Sort     string   `json:"sort" schema:"sort,default:popular"`
	Page     int      `json:"page" schema:"page"`
	PageSize int      `json:"pageSize" schema:"size,default:40"`
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

func queryFromRequest(r *http.Request, searchRequest *SearchRequest) error {

	err := json.NewDecoder(r.Body).Decode(searchRequest)
	if err != nil {
		return err
	}
	return nil
}

func queryFromRequestQuery(query url.Values, result *SearchRequest) error {
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	err := decoder.Decode(result, query)
	if err != nil {
		return err
	}

	for _, v := range query["str"] {

		parts := strings.Split(v, ":")
		if len(parts) != 2 {
			continue
		}
		id, err := strconv.Atoi(parts[0])

		if err != nil {
			continue
		}
		result.StringFilter = append(result.StringFilter, index.StringSearch{
			Id:    uint(id),
			Value: parts[1],
		})
	}

	for _, v := range query["num"] {
		var id uint
		var min float64
		var max float64
		_, err := fmt.Sscanf(v, "%d:%f-%f", &id, &min, &max)

		if err != nil {
			continue
		}
		result.NumberFilter = append(result.NumberFilter, index.NumberSearch[float64]{
			Id: uint(id),
			NumberRange: facet.NumberRange[float64]{
				Min: min,
				Max: max,
			},
		})
	}

	for _, v := range query["int"] {
		var id uint
		var min int
		var max int
		_, err := fmt.Sscanf(v, "%d:%d-%d", &id, &min, &max)

		if err != nil {
			continue
		}
		result.IntegerFilter = append(result.IntegerFilter, index.NumberSearch[int]{
			Id: uint(id),
			NumberRange: facet.NumberRange[int]{
				Min: min,
				Max: max,
			},
		})
	}

	return nil
}
