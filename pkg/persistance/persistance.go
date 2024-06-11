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
	Items map[int]index.DataItem
}

type FreeTextStorage struct {
	Documents map[int]search.Document
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

	return nil
}

func cloneFields(f map[int]index.ItemKeyField) map[int]string {
	fields := make(map[int]string)
	for k, v := range f {
		fields[k] = v.Value
	}
	return fields
}

func cloneNumberFields[K facet.FieldNumberValue](f map[int]index.ItemNumberField[K]) map[int]K {
	fields := make(map[int]K)
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
	fields := make(map[int]facet.KeyField)

	for _, fld := range idx.KeyFacets {
		fields[fld.Id] = *fld
	}

	items := make(map[int]index.DataItem)
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
	return nil
}
