package index

type Item struct {
	Id           int64             `json:"id"`
	Sku          string            `json:"sku"`
	Title        string            `json:"title"`
	Props        map[string]string `json:"props"`
	Fields       map[int64]string  `json:"values"`
	NumberFields map[int64]float64 `json:"numberValues"`
}

type Sort struct {
	FieldId int64 `json:"fieldId"`
	Asc     bool  `json:"asc"`
}
