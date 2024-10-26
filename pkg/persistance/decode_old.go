package persistance

import (
	"encoding/gob"

	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/types"
)

type KeyFieldValue struct {
	Value string `json:"value"`
	Id    uint   `json:"id"`
}

type DecimalFieldValue struct {
	Value float64 `json:"value"`
	Id    uint    `json:"id"`
}

type IntegerFieldValue struct {
	Value int  `json:"value"`
	Id    uint `json:"id"`
}

type ItemFields struct {
	Fields        []KeyFieldValue     `json:"values"`
	DecimalFields []DecimalFieldValue `json:"numberValues"`
	IntegerFields []IntegerFieldValue `json:"integerValues"`
}
type StoredItem struct {
	index.BaseItem
	ItemFields
}

func decodeOld(enc *gob.Decoder, item *index.DataItem) error {
	tmp := &StoredItem{}
	err := enc.Decode(tmp)
	if err == nil {
		fields := make(types.ItemFields)
		for _, field := range tmp.Fields {
			fields[field.Id] = field.Value
		}
		for _, field := range tmp.DecimalFields {
			fields[field.Id] = field.Value
		}
		for _, field := range tmp.IntegerFields {
			fields[field.Id] = field.Value
		}
		item.BaseItem = &tmp.BaseItem
		item.Fields = fields
	}
	return err
}
