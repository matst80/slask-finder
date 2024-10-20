package facet

import "tornberg.me/facet-search/pkg/types"

type Bucket[V FieldNumberValue] struct {
	values map[V]types.ItemList
}

func (b *Bucket[V]) AddValueLink(value V, item types.Item) {
	idList, ok := b.values[value]

	if !ok {
		b.values[value] = types.ItemList{item.GetId(): item}

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
	//delete(*b.all, id)
}

func MakeBucket[V FieldNumberValue](value V, item types.Item) Bucket[V] {
	return Bucket[V]{
		values: map[V]types.ItemList{value: {item.GetId(): item}},
	}
}

const Bits_To_Shift = 9

func GetBucket[V float64 | int | float32](value V) int {
	r := int(value) >> Bits_To_Shift
	return r
}
