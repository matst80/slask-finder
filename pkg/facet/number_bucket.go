package facet

type Bucket[V FieldNumberValue] struct {
	values map[V]ItemList
	//	all    *MatchList
}

func (b *Bucket[V]) AddValueLink(value V, item Item) {
	idList, ok := b.values[value]

	if !ok {
		b.values[value] = ItemList{item.GetId(): &item}

	} else {
		idList.Add(item)
	}
	//maps.Copy(*b.all, lst)
	//b.all.Merge(&lst)
}

func (b *Bucket[V]) RemoveValueLink(value V, id uint) {
	idList, ok := b.values[value]
	if !ok {
		return
	}
	delete(idList, id)
	//delete(*b.all, id)
}

func MakeBucket[V FieldNumberValue](value V, item Item) Bucket[V] {
	return Bucket[V]{
		values: map[V]ItemList{value: {item.GetId(): &item}},
		//		all:    &MatchList{id: fields},
	}
}

const Bits_To_Shift = 9

func GetBucket[V float64 | int | float32](value V) int {
	r := int(value) >> Bits_To_Shift
	return r
}
