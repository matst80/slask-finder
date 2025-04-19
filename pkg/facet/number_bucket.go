package facet

import "github.com/matst80/slask-finder/pkg/types"

type Bucket[V FieldNumberValue] struct {
	minValue V
	maxValue V
	values   map[V]types.ItemList
}

func (b *Bucket[V]) AddValueLink(value V, itemId uint) {
	idList, ok := b.values[value]
	if b.minValue > value {
		b.minValue = value
	}
	if b.maxValue < value {
		b.maxValue = value
	}
	if !ok {
		b.values[value] = types.ItemList{itemId: struct{}{}}

	} else {
		idList.AddId(itemId)
	}
}

func (b *Bucket[V]) RemoveValueLink(value V, id uint) {
	idList, ok := b.values[value]
	if !ok {
		return
	}
	delete(idList, id)
}

func MakeBucket[V FieldNumberValue](value V, itemId uint) Bucket[V] {
	return Bucket[V]{
		values: map[V]types.ItemList{value: {itemId: struct{}{}}},
	}
}

const Bits_To_Shift = 9

func GetBucket[V float64 | int | float32](value V) int {
	r := int(value) >> Bits_To_Shift
	return r
}
