package persistance

import (
	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
)

func cloneFields(f map[uint]index.ItemKeyField) map[uint]string {
	fields := make(map[uint]string)
	for k, v := range f {
		fields[k] = *v.Value
	}
	return fields
}

func cloneNumberFields[K facet.FieldNumberValue](f map[uint]index.ItemNumberField[K]) map[uint]K {
	fields := make(map[uint]K)
	for k, v := range f {
		fields[k] = v.Value
	}
	return fields
}
