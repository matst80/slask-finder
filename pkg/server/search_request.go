package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/gorilla/schema"
	"tornberg.me/facet-search/pkg/index"
)

type SearchRequest struct {
	*index.Filters
	Stock    []string `json:"stock" schema:"stock"`
	Query    string   `json:"query" schema:"query"`
	Sort     string   `json:"sort" schema:"sort,default:popular"`
	Page     int      `json:"page" schema:"page"`
	PageSize int      `json:"pageSize" schema:"size,default:40"`
}

func QueryFromRequest(r *http.Request) (SearchRequest, error) {
	sr := SearchRequest{
		Page:     0,
		PageSize: 25,
	}
	err := json.NewDecoder(r.Body).Decode(&sr)
	if err != nil {
		return sr, err
	}
	return sr, nil
}

func QueryFromRequestQuery(query url.Values, result *SearchRequest) error {
	decoder := schema.NewDecoder()
	decoder.RegisterConverter("str", func(value string) reflect.Value {
		parts := strings.Split(value, ":")
		if len(parts) != 2 {
			return reflect.Value{}
		}
		id, err := strconv.Atoi(parts[0])
		if err != nil {
			return reflect.Value{}
		}
		return reflect.ValueOf(index.StringSearch{
			Id:    uint(id),
			Value: parts[1],
		})
	})
	decoder.RegisterConverter("int", func(value string) reflect.Value {
		parts := strings.Split(value, ":")
		if len(parts) != 3 {
			return reflect.Value{}
		}
		id, err := strconv.Atoi(parts[0])
		if err != nil {
			return reflect.Value{}
		}
		min, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return reflect.Value{}
		}
		max, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return reflect.Value{}
		}
		return reflect.ValueOf(index.NumberSearch[float64]{
			Id:  uint(id),
			Min: min,
			Max: max,
		})
	})
	decoder.RegisterConverter("num", func(value string) reflect.Value {
		parts := strings.Split(value, ":")
		if len(parts) != 3 {
			return reflect.Value{}
		}
		id, err := strconv.Atoi(parts[0])
		if err != nil {
			return reflect.Value{}
		}
		min, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return reflect.Value{}
		}
		max, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return reflect.Value{}
		}
		return reflect.ValueOf(index.NumberSearch[float64]{
			Id:  uint(id),
			Min: min,
			Max: max,
		})
	})
	return decoder.Decode(result, query)
}
