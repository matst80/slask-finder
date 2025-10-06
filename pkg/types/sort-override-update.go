package types

type SortOverrideUpdate struct {
	Key  string             `json:"key"`
	Data map[uint32]float64 `json:"data"`
}
