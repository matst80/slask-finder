package facet

type Result struct {
	ids      map[int64]struct{}
	hasItems bool
}

func NewResult() Result {
	return Result{ids: make(map[int64]struct{})}
}

func (r *Result) Add(ids ...int64) {
	if len(ids) > 0 {
		r.hasItems = true
	}
	for _, id := range ids {
		r.ids[id] = struct{}{}
	}
}

func (r *Result) HasItems() bool {
	return r.hasItems
}

func (r *Result) GetMap() map[int64]struct{} {
	return r.ids
}

func (r *Result) Ids() []int64 {
	ids := make([]int64, len(r.ids))
	idx := 0
	for id := range r.ids {
		ids[idx] = id
		idx++
	}
	return ids
}

func (r *Result) SortedIds(srt SortIndex, maxItems int) []int64 {
	return srt.SortMap(r.ids, maxItems)
}

func (a *Result) Merge(b Result) {
	for id := range b.ids {
		a.ids[id] = struct{}{}
	}
}

func (a *Result) Length() int {
	return len(a.ids)
}

func (a *Result) Intersect(b Result) {
	l_a := len(a.ids)
	l_b := len(b.ids)
	if l_a == 0 {
		return
	}
	if l_b == 0 {
		a.ids = map[int64]struct{}{}
		return
	}
	for id := range a.ids {
		_, ok := b.ids[id]
		if !ok {
			delete(a.ids, id)
		}
	}
}

func (r *Result) Contains(id int64) bool {
	_, ok := r.ids[id]
	return ok
}

func MakeIntersectResult(r chan Result, len int) Result {
	if len == 0 {
		return Result{}
	}
	first := <-r
	for i := 1; i < len; i++ {
		first.Intersect(<-r)
	}
	return first
}
