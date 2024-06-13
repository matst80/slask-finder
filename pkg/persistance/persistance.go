package persistance

import (
	"encoding/gob"
	"io"
	"os"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
)

func NewPersistance() *Persistance {
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}(nil))
	return &Persistance{
		File:         "data/index.db",
		FreeTextFile: "data/freetext.db",
	}
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

	for _, item := range v.Items {
		idx.UpsertItem(item)
	}
	v = IndexStorage{}
	enc = nil
	return nil
}

func (p *Persistance) SaveIndex(idx *index.Index) error {

	file, err := os.Create(p.File)
	if err != nil {
		return err
	}
	fields := make(map[uint]facet.KeyField)

	for _, fld := range idx.KeyFacets {
		fields[fld.Id] = *fld
	}

	items := make(map[uint]index.DataItem)
	for _, item := range idx.Items {
		items[item.Id] = index.DataItem{
			BaseItem:      item.BaseItem,
			Fields:        cloneFields(item.Fields),
			DecimalFields: cloneNumberFields(item.DecimalFields),
			IntegerFields: cloneNumberFields(item.IntegerFields),
		}
	}

	defer file.Close()
	writer := io.Writer(file)
	enc := gob.NewEncoder(writer)
	err = enc.Encode(IndexStorage{
		Items: items,
	})
	if err != nil {
		return err
	}
	enc = nil
	items = nil
	return nil
}
