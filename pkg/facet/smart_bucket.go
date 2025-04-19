package facet

import "github.com/matst80/slask-finder/pkg/types"

type SmartBucket[V FieldNumberValue] struct {
	next     *SmartBucket[V]
	prev     *SmartBucket[V]
	maxSize  V
	MinValue V
	MaxValue V
	Ids      *types.ItemList
	Count    uint
}

func NewSmartBucket[V FieldNumberValue](maxSize V) *SmartBucket[V] {
	return &SmartBucket[V]{
		maxSize:  maxSize,
		Ids:      &types.ItemList{},
		MinValue: 0,
		MaxValue: 0,
		Count:    0,
	}
}

func (b *SmartBucket[V]) canFit(value V) bool {
	if value >= b.MaxValue && value <= b.MaxValue {
		return true
	}
	diff := b.MaxValue - b.MinValue
	if diff < b.maxSize {
		if value >= b.MinValue+b.maxSize && value <= b.MaxValue-b.maxSize {
			return true
		}

	}
	return false
}

func (b *SmartBucket[V]) findOrCreateBucket(value V) *SmartBucket[V] {
	if b.canFit(value) {
		return b
	}

	if value > b.MaxValue {
		if b.next == nil {
			b.next = NewSmartBucket(b.maxSize)
			b.next.MinValue = value
			b.next.MaxValue = value
			b.next.prev = b
			return b.next
		}
		if b.next.canFit(value) {
			return b.next
		} else {
			if b.next.MinValue > value {
				n := NewSmartBucket(b.maxSize)
				n.MinValue = value
				n.MaxValue = value
				// insert n between b and b.next
				n.next = b.next
				n.prev = b

				b.next.prev = n
				b.next = n
				return n
			} else {
				return b.next.findOrCreateBucket(value)
			}

		}

		//return b.next.findBucket(value)
	}
	if value < b.MinValue {
		if b.prev == nil {
			b.prev = NewSmartBucket(b.maxSize)
			b.prev.MinValue = value
			b.prev.MaxValue = value
			b.prev.next = b
			return b.prev
		}
		if b.prev.canFit(value) {
			return b.prev
		} else {

			if b.prev.MaxValue < value {
				n := NewSmartBucket(b.maxSize)
				n.MinValue = value
				n.MaxValue = value
				// insert n between b.prev and b
				n.prev = b.prev
				n.next = b
				b.prev.next = n
				b.prev = n
				return n
			} else {
				return b.prev.findOrCreateBucket(value)
			}

		}

	}

	return nil
}

func (b *SmartBucket[V]) AddValueLink(value V, itemId uint) {

	target := b.findOrCreateBucket(value)
	if target.Count == 0 {
		target.MinValue = value
		target.MaxValue = value
	} else {
		if value < target.MinValue {
			target.MinValue = value
		}
		if value > target.MaxValue {
			target.MaxValue = value
		}
	}
	target.Count++
	target.Ids.AddId(itemId)

}

func (b *SmartBucket[V]) RemoveValueLink(value V, id uint) {
	target := b.findOrCreateBucket(value)
	if target == nil {
		return
	}
	delete(*target.Ids, id)
	target.Count--
	if target.Count == 0 {
		if target.prev != nil {
			target.prev.next = target.next
		}
		if target.next != nil {
			target.next.prev = target.prev
		}
	}
}

func (b *SmartBucket[V]) TotalCount() int {
	count := 0
	curr := b
	for curr != nil {
		count += int(curr.Count)
		curr = curr.next
	}
	return count
}

func (b *SmartBucket[V]) findBucket(value V) *SmartBucket[V] {
	curr := b
	for curr != nil {
		if curr.MinValue <= value && curr.MaxValue >= value {
			return curr
		}
		curr = curr.next
	}
	return nil
}

func (b *SmartBucket[V]) Match(min, max V) types.ItemList {
	ret := make(types.ItemList)
	curr := b.findBucket(min)
	for curr != nil {
		if curr.Count > 0 {
			ret.Merge(curr.Ids)
		}
		if curr.next == nil || curr.next.MinValue >= max {
			break
		}
		curr = curr.next
	}
	return ret
}
