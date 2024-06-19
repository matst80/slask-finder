package facet

type NumberBucket struct {
	IdList
}

type Bucket[V FieldNumberValue] struct {
	values map[V]MatchList
	//	all    *MatchList
}

func (b *Bucket[V]) AddValueLink(value V, id uint, fields *ItemFields) {
	idList, ok := b.values[value]
	lst := MatchList{id: fields}
	if !ok {
		b.values[value] = lst

	} else {
		idList[id] = fields
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

func MakeBucket[V FieldNumberValue](value V, id uint, fields *ItemFields) Bucket[V] {
	return Bucket[V]{
		values: map[V]MatchList{value: {id: fields}},
		//		all:    &MatchList{id: fields},
	}
}

func MakeBucketList[V FieldNumberValue](value V, ids *MatchList) Bucket[V] {
	return Bucket[V]{
		values: map[V]MatchList{value: *ids},
		//all:    ids,
	}
}

const Bits_To_Shift = 9

func GetBucket[V float64 | int | float32](value V) int {
	r := int(value) >> Bits_To_Shift
	return r
}
