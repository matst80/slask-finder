package types

type SortOverrideUpdate struct {
	Key  string           `json:"key"`
	Data map[uint]float64 `json:"data"`
}
