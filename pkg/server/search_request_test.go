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
		FacetRequest: &types.FacetRequest{
			IgnoreFacets: []uint{},
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
	if sr.StringFilter[0].Id != 10 || sr.StringFilter[0].Value[0] != "tio" {
		t.Errorf("Expected string filter to be [10:tio], got %v", sr.StringFilter)
	}
	if sr.RangeFilter[0].Id != 12 || sr.RangeFilter[0].Min != 12.0 || sr.RangeFilter[0].Max != 12.0 {
		t.Errorf("Expected range filter to be [12:12-12], got %v", sr.RangeFilter[0])
	}
	if sr.RangeFilter[1].Id != 14 || sr.RangeFilter[1].Min != 14.0 || sr.RangeFilter[1].Max != 14.0 {
		t.Errorf("Expected range filter to be [14:14-14], got %v", sr.RangeFilter[1])
	}
}

func TestRealUrl(t *testing.T) {
	realUrl := "https://slask-finder.knatofs.se/api/stream?page=0&size=120&sort=popular&query=&rng=36307%3A65-9999&str=32077%3ASocket+AM5&str=31158%3ACPU+AIO-vattenkylare%7C%7CVattenkylning&str=30303%3A!hej"
	parsedUrl, err := url.Parse(realUrl)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}
	sr := SearchRequest{
		Page:     0,
		PageSize: 120,
		FacetRequest: &types.FacetRequest{
			Filters: &types.Filters{
				StringFilter: []types.StringFilter{},
				RangeFilter:  []types.RangeFilter{},
			},
			Stock: []string{},
			Query: "",
		},
		Sort: "popular",
	}
	queryFromRequestQuery(parsedUrl.Query(), &sr)
	t.Logf("%+v", sr)
	if sr.FacetRequest.Filters.StringFilter[0].Value[0] != "Socket AM5" {
		t.Errorf("Expected string filter to be [32077:Socket AM5], got %v", sr.FacetRequest.Filters.StringFilter[0])
	}
	if sr.FacetRequest.Filters.StringFilter[1].Value[0] != "CPU AIO-vattenkylare" {
		t.Errorf("Expected string filter to be [31158:CPU AIO-vattenkylare], got %v", sr.FacetRequest.Filters.StringFilter[1])
	}
	if sr.FacetRequest.Filters.StringFilter[1].Value[1] != "Vattenkylning" {
		t.Errorf("Expected string filter to be [31158:CPU AIO-vattenkylare], got %v", sr.FacetRequest.Filters.StringFilter[1])
	}
	if sr.FacetRequest.Filters.RangeFilter[0].Id != 36307 {
		t.Errorf("Expected range filter to be [36307:65-9999], got %v", sr.FacetRequest.Filters.RangeFilter[0])
	}
	if sr.FacetRequest.Filters.RangeFilter[0].Min != 65.0 {
		t.Errorf("Expected range filter to be [36307:65-9999], got %v", sr.FacetRequest.Filters.RangeFilter[0])
	}
	if sr.FacetRequest.Filters.RangeFilter[0].Max != 9999.0 {
		t.Errorf("Expected range filter to be [36307:65-9999], got %v", sr.FacetRequest.Filters.RangeFilter[0])
	}
	if sr.FacetRequest.Filters.StringFilter[2].Value[0] != "hej" {
		t.Errorf("Expected string filter to be [30303:hej], got %v", sr.FacetRequest.Filters.StringFilter[2])
	}
	if sr.FacetRequest.Filters.StringFilter[2].Not != true {
		t.Errorf("Expected string filter to be [30303:hej], got %v", sr.FacetRequest.Filters.StringFilter[2])
	}
	if sr.FacetRequest.Filters.StringFilter[2].Id != 30303 {
		t.Errorf("Expected string filter to be [30303:hej], got %v", sr.FacetRequest.Filters.StringFilter[2])
	}
}
