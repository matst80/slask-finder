package types

import "maps"

type ItemList map[uint]struct{}

func (i *ItemList) Add(item Item) {
	(*i)[item.GetId()] = struct{}{}
}

func (i *ItemList) AddId(id uint) {
	(*i)[id] = struct{}{}
}

func (a ItemList) Intersect(b ItemList) {
	for id := range a {
		_, ok := b[id]
		if !ok {
			delete(a, id)
		}
	}
}

func Intersect[K any](a ItemList, b map[uint]K) ItemList {
	result := make(ItemList)
	for id := range a {
		if _, ok := b[id]; ok {
			result[id] = struct{}{}
		}
	}
	return result
}

func Merge[K any](a ItemList, b map[uint]K) {
	for id := range b {
		a[id] = struct{}{}
	}
}

func (i ItemList) Merge(other *ItemList) {
	maps.Copy(i, *other)
}

func (i ItemList) HasIntersection(other *ItemList) bool {
	for id := range i {
		_, ok := (*other)[id]
		if ok {
			return true
		}
	}
	return false
}

func MakeIntersectResult(r chan *ItemList, len int) *ItemList {
	defer close(r)
	first := &ItemList{}
	if len == 0 {
		return first
	}

	next := <-r
	if next != nil {
		first.Merge(next)
	}

	for i := 1; i < len; i++ {
		next = <-r
		if next != nil {
			first.Intersect(*next)
		} else {
			return &ItemList{}
		}
	}

	return first
}
