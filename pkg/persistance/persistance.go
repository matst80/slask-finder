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
		File:         "data/index-v2.db",
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

	// read items from reader and store them in the index

	for err == nil {
		var v index.DataItem
		err = enc.Decode(&v)
		if err == nil {
			idx.UpsertItem(v)
		}
	}
	if err.Error() == "EOF" {
		err = nil
	}
	return err
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

	//items := make(map[uint]index.DataItem)

	defer file.Close()
	writer := io.Writer(file)
	enc := gob.NewEncoder(writer)

	for _, item := range idx.Items {
		err = enc.Encode(index.DataItem{
			BaseItem:      item.BaseItem,
			Fields:        cloneFields(item.Fields),
			DecimalFields: cloneNumberFields(item.DecimalFields),
			IntegerFields: cloneNumberFields(item.IntegerFields),
		})
		if err != nil {
			return err
		}

	}

	enc = nil

	return nil
}
