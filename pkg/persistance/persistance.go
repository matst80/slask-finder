package persistance

import (
	"encoding/gob"
	"io"
	"os"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
	"tornberg.me/facet-search/pkg/search"
)

type Persistance struct {
	File         string
	FreeTextFile string
}

func NewPersistance() Persistance {
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}(nil))
	return Persistance{
		File:         "data/index.db",
		FreeTextFile: "data/freetext.db",
	}
}

type IndexStorage struct {
	Items map[uint]index.DataItem
}

type FreeTextStorage struct {
	Documents map[uint]search.Document
}

func (p *Persistance) LoadFreeText(ft *search.FreeTextIndex) error {
	file, err := os.Open(p.FreeTextFile)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := io.Reader(file)
	enc := gob.NewDecoder(reader)
	var v FreeTextStorage
	err = enc.Decode(&v)
	if err != nil {
		return err
	}

	ft.Documents = v.Documents
	enc = nil
	return nil
}

func (p *Persistance) SaveFreeText(ft *search.FreeTextIndex) error {
	file, err := os.Create(p.FreeTextFile)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := io.Writer(file)
	enc := gob.NewEncoder(writer)
	err = enc.Encode(FreeTextStorage{
		Documents: ft.Documents,
	})
	if err != nil {
		return err
	}

	return nil
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
		idx.AddItem(item)
	}
	v = IndexStorage{}
	enc = nil
	return nil
}

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
