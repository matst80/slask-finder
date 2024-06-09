package facet

type ValueFieldReference[V FieldKeyValue] struct {
	Value V     `json:"value"`
	Id    int64 `json:"id"`
}

type NumberFieldReference[V FieldNumberValue] struct {
	Value V     `json:"value"`
	Id    int64 `json:"id"`
}
