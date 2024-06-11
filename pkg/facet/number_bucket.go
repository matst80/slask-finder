package facet

type NumberBucket struct {
	IdList
}

type Bucket[V FieldNumberValue] struct {
	values map[V]IdList
	all    *IdList
}

func (b *Bucket[V]) AddValueLink(value V, id int64) {
	idList, ok := b.values[value]
	lst := IdList{id: struct{}{}}
	if !ok {
		b.values[value] = lst

	} else {
		idList[id] = struct{}{}
	}
	b.all.Merge(&lst)
}

func MakeBucket[V FieldNumberValue](value V, id int64) Bucket[V] {
	return Bucket[V]{
		values: map[V]IdList{value: {id: struct{}{}}},
		all:    &IdList{id: struct{}{}},
	}
}

func MakeBucketList[V FieldNumberValue](value V, ids *IdList) Bucket[V] {
	return Bucket[V]{
		values: map[V]IdList{value: *ids},
		all:    ids,
	}
}

const Bits_To_Shift = 7

func GetBucket[V float64 | int64 | int | float32](value V) int64 {
	r := int64(value) >> Bits_To_Shift
	return r
}
