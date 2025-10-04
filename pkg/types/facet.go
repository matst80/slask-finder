package types

type FieldType uint

type StorageFacet struct {
	*BaseField
	Type FieldType `json:"type"`
}
