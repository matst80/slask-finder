package types

type ItemFacets interface {
	SetValue(id uint, value any)
	GetFacetValue(facetId uint) (any, bool)
	GetFacets() map[uint]any
}

type ItemFields map[uint]any

func (b ItemFields) GetFacetValue(id uint) (any, bool) {
	v, ok := b[id]
	return v, ok
}

func (b ItemFields) GetFacets() map[uint]any {
	return b
}

func (b ItemFields) SetValue(id uint, value any) {
	b[id] = value
}
