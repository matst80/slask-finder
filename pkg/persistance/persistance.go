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
	gob.Register([]interface{}(nil))
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
	for _, fld := range v.Fields {
		idx.AddField(fld)
	}
	for _, fld := range v.NumberFields {
		idx.AddNumberField(fld)
	}
	for _, item := range v.Items {
		idx.AddItem(item)
	}
	s := index.MakeSortFromNumberField(idx.Items, 4)
	idx.Sort = &s
	return nil
}

func (p *Persistance) SaveIndex(idx *index.Index) error {

	file, err := os.Create(p.File)
	if err != nil {
		return err
	}
	fields := make(map[int64]facet.Field)
	numberFields := make(map[int64]facet.Field)
	for _, fld := range idx.Fields {
		fields[fld.Field.Id] = fld.Field
	}
	for _, fld := range idx.NumberFields {
		numberFields[fld.Field.Id] = fld.Field
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
