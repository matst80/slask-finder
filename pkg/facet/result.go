package facet

type Result struct {
	Ids []int64
}

func (r *Result) Add(ids ...int64) {
	for _, id := range ids {
		found := false
		for _, existing := range r.Ids {
			if existing == id {
				found = true
			}
		}
		if !found {
			r.Ids = append(r.Ids, id)
		}
	}
}

func (a *Result) Merge(b Result) {
	a.Add(b.Ids...)
}

func (r *Result) Intersect(b Result) {
	var ids []int64
	if len(b.Ids) == 0 {
		r.Ids = []int64{}
	}
	for _, id := range r.Ids {
		for _, id2 := range b.Ids {
			if id == id2 {
				ids = append(ids, id)
			}
		}
	}
	r.Ids = ids
}

func (r *Result) Contains(id int64) bool {
	for _, existing := range r.Ids {
		if existing == id {
			return true
		}
	}
	return false
}
