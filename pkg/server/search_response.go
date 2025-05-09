package server

type SearchResponse struct {
	Duration  string `json:"duration"`
	Page      int    `json:"page"`
	PageSize  int    `json:"pageSize"`
	Sort      string `json:"sort"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
	TotalHits int    `json:"totalHits"`
}
