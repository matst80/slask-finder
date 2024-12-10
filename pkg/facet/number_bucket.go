package facet

import "github.com/matst80/slask-finder/pkg/types"

type Bucket[V FieldNumberValue] struct {
	values map[V]types.ItemList
}

func (b *Bucket[V]) AddValueLink(value V, item types.Item) {
	idList, ok := b.values[value]

	if !ok {
		b.values[value] = types.ItemList{item.GetId(): struct{}{}}

	} else {
		idList.Add(item)
	}
}

func (b *Bucket[V]) RemoveValueLink(value V, id uint) {
	idList, ok := b.values[value]
	if !ok {
		return
	}
	delete(idList, id)
}

func MakeBucket[V FieldNumberValue](value V, item types.Item) Bucket[V] {
	return Bucket[V]{
		values: map[V]types.ItemList{value: {item.GetId(): struct{}{}}},
	}
}

const Bits_To_Shift = 9

func GetBucket[V float64 | int | float32](value V) int {
	r := int(value) >> Bits_To_Shift
	return r
}
