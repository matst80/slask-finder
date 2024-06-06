package index

import "tornberg.me/facet-search/pkg/facet"

type Item struct {
	Id           int64                        `json:"id"`
	Title        string                       `json:"title"`
	Props        map[string]string            `json:"props"`
	Fields       []facet.StringFieldReference `json:"values"`
	NumberFields []facet.NumberFieldReference `json:"numberValues"`
}
