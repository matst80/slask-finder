package facet

type NumberValueField struct {
	Field  `json:"field"`
	values map[float64][]int64
}

func (f *NumberValueField) Matches(min float64, max float64) Result {
	result := Result{}
	if f.values == nil {
		return result
	}
	for v, ids := range f.values {
		if v >= min && v <= max {
			result.Add(ids...)
		}
	}

	return result
}

type NumberRange struct {
	Min    float64   `json:"min"`
	Max    float64   `json:"max"`
	Values []float64 `json:"values"`
}

func (f *NumberValueField) Values() NumberRange {
	values := []float64{}
	min := 99999999999999999.0
	max := -99999999999999999.0
	for value := range f.values {
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
		values = append(values, value)
	}

	return NumberRange{Min: min, Max: max, Values: values}
}

func (f *NumberValueField) AddValueLink(value float64, ids ...int64) {
	if f.values == nil {
		f.values = make(map[float64][]int64)
	}
	f.values[value] = append(f.values[value], ids...)
}

func NewNumberValueField(field Field, value float64, ids ...int64) NumberValueField {
	return NumberValueField{
		Field:  field,
		values: map[float64][]int64{value: ids},
	}
}

func EmptyNumberField(field Field) NumberValueField {
	return NumberValueField{
		Field:  field,
		values: map[float64][]int64{},
	}
}
