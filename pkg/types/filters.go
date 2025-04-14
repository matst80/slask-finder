package types

type StringFilter struct {
	Id    uint        `json:"id"`
	Value interface{} `json:"value"`
}

type RangeFilter struct {
	Min interface{} `json:"min"`
	Max interface{} `json:"max"`
	Id  uint        `json:"id"`
}

type FilterIds map[uint]struct{}

type Filters struct {
	ids          *FilterIds
	StringFilter []StringFilter `json:"string" schema:"-"`
	RangeFilter  []RangeFilter  `json:"range" schema:"-"`
}

func (f *Filters) WithOut(id uint, dontExclude bool) *Filters {
	if dontExclude {
		return f
	}
	result := Filters{
		StringFilter: make([]StringFilter, 0, len(f.StringFilter)),
		RangeFilter:  make([]RangeFilter, 0, len(f.RangeFilter)),
	}
	for _, filter := range f.StringFilter {
		if filter.Id != id {
			result.StringFilter = append(result.StringFilter, filter)
		}
	}
	for _, filter := range f.RangeFilter {
		if filter.Id != id {
			result.RangeFilter = append(result.RangeFilter, filter)
		}
	}
	return &result
}

func (f *Filters) getIds() *FilterIds {
	if f.ids == nil {
		ids := make(FilterIds)
		if f.StringFilter != nil {
			for _, filter := range f.StringFilter {
				ids[filter.Id] = struct{}{}
			}
		}
		if f.RangeFilter != nil {
			for _, filter := range f.RangeFilter {
				ids[filter.Id] = struct{}{}
			}
		}
		f.ids = &ids
	}
	return f.ids
}

func (f *Filters) HasField(id uint) bool {
	ids := f.getIds()
	_, ok := (*ids)[id]
	return ok
}

// func (f *Filters) HasCategoryFilter() bool {
// 	return slices.ContainsFunc(f.StringFilter, func(filter StringFilter) bool {
// 		return filter.Id >= 30 && filter.Id <= 35 && filter.Id != 23
// 	})
// }
