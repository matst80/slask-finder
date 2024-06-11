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
	// KeyFacets     map[int64]facet.Field[string]
	// DecimalFacets map[int64]facet.NumberField[float64]
	// IntFacets     map[int64]facet.NumberField[int]
	// BoolFacets    map[int64]facet.Field[bool]
	Items map[int64]index.DataItem
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
	// for _, fld := range v.KeyFacets {
	// 	idx.AddKeyField(&fld)
	// }
	// for _, fld := range v.DecimalFacets {
	// 	idx.AddDecimalField(&fld)
	// }
	// for _, fld := range v.IntFacets {
	// 	idx.AddIntegerField(&fld)
	// }
	// for _, fld := range v.BoolFacets {
	// 	idx.AddBoolField(&fld)
	// }
	for _, item := range v.Items {
		idx.AddItem(item)
	}
	//s := index.MakeSortFromNumberField(idx.Items, 4)
	//idx.Sort = &s
	return nil
}

func cloneFields[K facet.FieldKeyValue](f map[int64]index.ItemKeyField[K]) map[int64]K {
	fields := make(map[int64]K)
	for k, v := range f {
		fields[k] = v.Value
	}
	return fields
}

func cloneNumberFields[K facet.FieldNumberValue](f map[int64]index.ItemNumberField[K]) map[int64]K {
	fields := make(map[int64]K)
	for k, v := range f {
		fields[k] = v.Value
	}
	return fields
}

// func cloneField[T facet.FieldValue](f map[int64]*facet.KeyField[T]) map[int64]facet.KeyField[T] {
// 	fields := make(map[int64]facet.KeyField[T])
// 	for _, fld := range f {
// 		fields[fld.Id] = *fld
// 	}
// 	return fields
// }

// func cloneNumberField[T facet.FieldNumberValue](f map[int64]*facet.NumberField[T]) map[int64]facet.NumberField[T] {
// 	fields := make(map[int64]facet.NumberField[T])
// 	for _, fld := range f {
// 		fields[fld.Id] = *fld
// 	}
// 	return fields
// }

func (p *Persistance) SaveIndex(idx *index.Index) error {

	file, err := os.Create(p.File)
	if err != nil {
		return err
	}
	fields := make(map[int64]facet.KeyField[string])

	for _, fld := range idx.KeyFacets {
		fields[fld.Id] = *fld
	}

	items := make(map[int64]index.DataItem)
	for _, item := range idx.Items {
		items[item.Id] = index.DataItem{
			BaseItem:      item.BaseItem,
			Fields:        cloneFields(item.Fields),
			DecimalFields: cloneNumberFields(item.DecimalFields),
			IntegerFields: cloneNumberFields(item.IntegerFields),
			BoolFields:    cloneFields(item.BoolFields),
		}
	}

	toSave := IndexStorage{
		// KeyFacets:     cloneField(idx.KeyFacets),
		// DecimalFacets: cloneNumberField(idx.DecimalFacets),
		// IntFacets:     cloneNumberField(idx.IntFacets),
		// BoolFacets:    cloneField(idx.BoolFacets),
		Items: items,
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
