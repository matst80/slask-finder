package types

import "maps"

type ItemList map[uint]struct{}

func (a ItemList) Exclude(b *ItemList) {
	for id := range *b {
		_, ok := a[id]
		if ok {
			delete(a, id)
		}
	}
}

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

func (a ItemList) ToIntersected(b ItemList) (*ItemList, bool) {
	result := make(ItemList)

	for id := range a {
		if _, ok := b[id]; ok {
			result[id] = struct{}{}
		}
	}
	return &result, len(result) > 0
}

func (a ItemList) OnIntersect(b ItemList, onMatch func(id uint) bool) {
	al := len(a)
	bl := len(b)
	if al == 0 || bl == 0 {
		return
	}
	if al > bl {
		a, b = b, a
	}

	for id := range a {
		if _, ok := b[id]; ok {
			if !onMatch(id) {
				break
			}
		}
	}
}

// func Intersect[K any](a ItemList, b map[uint]K) ItemList {
// 	result := make(ItemList)
// 	for id := range a {
// 		if _, ok := b[id]; ok {
// 			result[id] = struct{}{}
// 		}
// 	}
// 	return result
// }

func Merge[K any](a ItemList, b map[uint]K) {
	for id := range b {
		a[id] = struct{}{}
	}
}

func (i ItemList) Merge(other *ItemList) {
	maps.Copy(i, *other)
}

func (i ItemList) HasIntersection(other *ItemList) bool {
	found := false
	i.OnIntersect(*other, func(id uint) bool {
		found = true
		return false
	})
	return found
	// l1 := len(i)
	// l2 := len(*other)
	// if l1 == 0 || l2 == 0 {
	// 	return false
	// }
	// for id := range i {
	// 	_, ok := (*other)[id]
	// 	if ok {
	// 		return true
	// 	}
	// }
	// return false
}

func (a ItemList) IntersectionLen(b ItemList) int {

	count := 0
	al := len(a)
	bl := len(b)
	if al == 0 || bl == 0 {
		return count
	}
	if al > bl {
		a, b = b, a
	}
	ok := false
	for id := range a {
		if _, ok = b[id]; ok {
			count++
		}
	}
	return count
}

type FilterResult struct {
	Ids    *ItemList
	Exlude bool
}

func MakeIntersectResult(r chan *FilterResult, len int) *ItemList {
	defer close(r)
	first := &ItemList{}
	if len == 0 {
		return first
	}

	next := <-r
	if next.Ids != nil {
		first.Merge(next.Ids)
	}

	for i := 1; i < len; i++ {
		next = <-r
		if next.Ids != nil {
			if next.Exlude {
				first.Exclude(next.Ids)
			} else {
				first.Intersect(*next.Ids)
			}
		} else {
			return &ItemList{}
		}
	}

	return first
}
