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

	if len == 0 {
		return &ItemList{}
	}
	if len == 1 {
		return <-r
	}
	first := ItemList{}
	first.Merge(<-r)
	//first := <-r
	for i := 1; i < len; i++ {
		first.Intersect(*<-r)
	}
	close(r)
	return &first
}
