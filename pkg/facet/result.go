package facet

type Result struct {
	ids map[int64]bool
}

func NewResult() Result {
	return Result{ids: make(map[int64]bool)}
}

func (r *Result) Add(ids ...int64) {
	for _, id := range ids {
		r.ids[id] = true
	}
	//r.Ids = append(r.Ids, ids...)
	// for _, id := range ids {
	// 	found := false
	// 	for _, existing := range r.Ids {
	// 		if existing == id {
	// 			found = true
	// 		}
	// 	}
	// 	if !found {
	// 		r.Ids = append(r.Ids, id)
	// 	}
	// }
}

func (r *Result) Ids() []int64 {
	var ids []int64
	for id, v := range r.ids {
		if v {
			ids = append(ids, id)
		}
	}
	return ids
}

func (a *Result) Merge(b Result) {
	for id := range b.ids {
		a.ids[id] = true
	}
}

func (a *Result) Intersect(b Result) {
	l_a := len(a.ids)
	l_b := len(b.ids)
	if l_a == 0 {
		return
	}
	if l_b == 0 {
		a.ids = map[int64]bool{}
		return
	}
	for id := range a.ids {
		if !b.ids[id] {
			a.ids[id] = false
		}
	}
}

func (r *Result) Contains(id int64) bool {
	v, ok := r.ids[id]
	return ok && v
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
