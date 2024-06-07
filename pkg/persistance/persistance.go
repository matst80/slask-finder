package persistance

import (
	"encoding/gob"
	"io"
	"os"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
)

type Persistance struct {
	File string
}

func NewPersistance() Persistance {
	return Persistance{
		File: "index.db",
	}
}

type IndexStorage struct {
	Fields       map[int64]facet.Field
	NumberFields map[int64]facet.Field
	Items        map[int64]index.Item
}

func (p *Persistance) LoadIndex(idx *index.Index) error {
	file, err := os.Open(p.File)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := io.Reader(file)
	enc := gob.NewDecoder(reader)
	var v IndexStorage
	err = enc.Decode(&v)

	if err != nil {
		return err
	}
	for k, fld := range v.Fields {
		idx.AddField(k, fld)
	}
	for k, fld := range v.NumberFields {
		idx.AddNumberField(k, fld)
	}
	for _, item := range v.Items {
		idx.AddItem(item)
	}

	return nil
}

func (p *Persistance) SaveIndex(idx *index.Index) error {
	file, err := os.Create(p.File)
	if err != nil {
		return err
	}
	fields := make(map[int64]facet.Field)
	numberFields := make(map[int64]facet.Field)
	for k, fld := range idx.Fields {
		fields[k] = fld.Field
	}
	for k, fld := range idx.NumberFields {
		numberFields[k] = fld.Field
	}

	toSave := IndexStorage{
		Fields:       fields,
		NumberFields: numberFields,
		Items:        idx.Items,
	}
	defer file.Close()
	writer := io.Writer(file)
	enc := gob.NewEncoder(writer)
	err = enc.Encode(toSave)
	if err != nil {
		return err
	}
	return nil
}
