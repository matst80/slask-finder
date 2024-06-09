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
	gob.Register([]interface{}(nil))
	return Persistance{
		File:         "data/index.db",
		FreeTextFile: "data/freetext.db",
	}
}

type IndexStorage struct {
	Fields       map[int64]facet.Field[string]
	NumberFields map[int64]facet.NumberField[float64]
	Items        map[int64]index.Item
}

type FreeTextStorage struct {
	Documents map[int64]search.Document
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
	for _, fld := range v.Fields {
		idx.AddField(fld)
	}
	for _, fld := range v.NumberFields {
		idx.AddNumberField(fld)
	}
	for _, item := range v.Items {
		idx.AddItem(item)
	}
	//s := index.MakeSortFromNumberField(idx.Items, 4)
	//idx.Sort = &s
	return nil
}

func (p *Persistance) SaveIndex(idx *index.Index) error {

	file, err := os.Create(p.File)
	if err != nil {
		return err
	}
	// fields := make(map[int64]facet.Field[string])
	// numberFields := make(map[int64]facet.NumberField[float64])
	// for _, fld := range idx.Fields {
	// 	fields[fld.Id] = fld
	// }
	// for _, fld := range idx.NumberFields {
	// 	numberFields[fld.Id] = fld
	// }

	toSave := IndexStorage{
		Fields:       idx.Fields,
		NumberFields: idx.NumberFields,
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
