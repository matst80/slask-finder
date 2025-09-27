package types

type ItemFacets interface {
	SetValue(id uint, value interface{})
	GetFacetValue(facetId uint) (interface{}, bool)
	GetFacets() map[uint]interface{}
}

type ItemFields map[uint]interface{}

func (b ItemFields) GetFacetValue(id uint) (interface{}, bool) {
	v, ok := b[id]
	return v, ok
}

func (b ItemFields) GetFacets() map[uint]interface{} {
	return b
}

func (b ItemFields) SetValue(id uint, value interface{}) {
	b[id] = value
}
