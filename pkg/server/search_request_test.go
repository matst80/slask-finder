package server

import (
	"net/url"
	"testing"

	"github.com/matst80/slask-finder/pkg/types"
)

func TestParseQueryValues(t *testing.T) {
	query := url.Values{
		"stock": []string{"1", "2"},
		"query": []string{"test"},
		"sort":  []string{"asc"},
		"page":  []string{"1"},
		"size":  []string{"10"},
		"str":   []string{"10:tio", "11:elva"},
		"rng":   []string{"12:12-12", "14:14-14"},
	}
	sr := &SearchRequest{
		Page:     0,
		PageSize: 25,
		FacetRequest: &FacetRequest{
			Filters: &types.Filters{
				StringFilter: []types.StringFilter{},
				RangeFilter:  []types.RangeFilter{},
			},
			Stock: []string{},
			Query: "",
		},
		Sort: "popular",
	}
	err := queryFromRequestQuery(query, sr)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if sr.Stock[0] != "1" || sr.Stock[1] != "2" {
		t.Errorf("Expected stock to be [1,2], got %v", sr.Stock)
	}
	if sr.Query != "test" {
		t.Errorf("Expected query to be test, got %v", sr.Query)
	}
	if sr.Sort != "asc" {
		t.Errorf("Expected sort to be asc, got %v", sr.Sort)
	}
	if sr.Page != 1 {
		t.Errorf("Expected page to be 1, got %v", sr.Page)
	}
	if sr.PageSize != 10 {
		t.Errorf("Expected page size to be 10, got %v", sr.PageSize)
	}
	if sr.StringFilter[0].Id != 10 || sr.StringFilter[0].Value != "tio" {
		t.Errorf("Expected string filter to be [10:tio], got %v", sr.StringFilter)
	}
	if sr.RangeFilter[0].Id != 12 || sr.RangeFilter[0].Min != 12.0 || sr.RangeFilter[0].Max != 12.0 {
		t.Errorf("Expected range filter to be [12:12-12], got %v", sr.RangeFilter[0])
	}
	if sr.RangeFilter[1].Id != 14 || sr.RangeFilter[1].Min != 14.0 || sr.RangeFilter[1].Max != 14.0 {
		t.Errorf("Expected range filter to be [14:14-14], got %v", sr.RangeFilter[1])
	}
}
