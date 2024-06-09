package facet

type Field struct {
	Id          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ValueField struct {
	Field  `json:"field"`
	values map[string][]int64
}

func (f *ValueField) Matches(search_strings ...string) Result {
	result := NewResult()
	for _, v := range search_strings {
		for key, ids := range f.values {
			if key == v {
				result.Add(ids...)
			}
		}
	}
	return result
}

func (f *ValueField) Values() []string {
	values := make([]string, len(f.values))

	i := 0
	for k := range f.values {
		values[i] = k
		i++
	}
	return values
}

func (f *ValueField) AddValueLink(value string, ids ...int64) {
	f.values[value] = append(f.values[value], ids...)
}

func (f *ValueField) TotalCount() int {
	total := 0
	for _, ids := range f.values {
		total += len(ids)
	}
	return total
}

func NewValueField(field Field, value string, ids ...int64) ValueField {
	return ValueField{
		Field:  field,
		values: map[string][]int64{value: ids},
	}
}

func EmptyValueField(field Field) ValueField {
	return ValueField{
		Field:  field,
		values: map[string][]int64{},
	}
}
