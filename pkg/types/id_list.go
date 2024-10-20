package types

type IdList map[uint]struct{}

var empty = struct{}{}

func (r *IdList) Add(id uint) {
	(*r)[id] = empty
}
