package facet

type StringFieldReference struct {
	Value string `json:"value"`
	Id    int64  `json:"id"`
}

type NumberFieldReference struct {
	Value float64 `json:"value"`
	Id    int64   `json:"id"`
}
