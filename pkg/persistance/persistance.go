package persistance

import (
	"compress/gzip"
	"encoding/gob"
	"os"
	"runtime"

	"tornberg.me/facet-search/pkg/facet"
	"tornberg.me/facet-search/pkg/index"
)

func NewPersistance() *Persistance {
	gob.Register(index.DataItem{})
	// gob.Register([]interface{}(nil))
	return &Persistance{
		File:         "data/index-v2.dbz",
		FreeTextFile: "data/freetext.dbz",
	}
}

// type KeyFieldValue struct {
// 	Value string `json:"value"`
// 	Id    uint   `json:"id"`
// }

// type DecimalFieldValue struct {
// 	Value float64 `json:"value"`
// 	Id    uint    `json:"id"`
// }

// type IntegerFieldValue struct {
// 	Value int  `json:"value"`
// 	Id    uint `json:"id"`
// }

// type ItemFields struct {
// 	Fields        []KeyFieldValue     `json:"values"`
// 	DecimalFields []DecimalFieldValue `json:"numberValues"`
// 	IntegerFields []IntegerFieldValue `json:"integerValues"`
// }
// type StoredItem struct {
// 	index.BaseItem
// 	ItemFields
// }

type Field struct {
	Id    uint
	Value interface{}
}

type PersistanceItem struct {
	Data   index.BaseItem
	Fields []Field
}

type PersistanceItem2 struct {
	Data   index.BaseItem
	Fields facet.ItemFields
}

func (p *Persistance) LoadIndex(idx *index.Index) error {

	file, err := os.Open(p.File)
	if err != nil {
		return err
	}
	defer runtime.GC()
	defer file.Close()

	zipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}

	enc := gob.NewDecoder(zipReader)
	defer zipReader.Close()

	idx.Lock()
	defer idx.Unlock()
	tmp := index.DataItem{}
	for err == nil {

		if err = enc.Decode(&tmp); err == nil {
			idx.UpsertItemUnsafe(tmp)
			tmp = index.DataItem{}
		}
	}
	//enc = nil
	//v = nil
	//tmp = nil
	if err.Error() == "EOF" {
		return nil
	}

	return err
}

func (p *Persistance) SaveIndex(idx *index.Index) error {

	file, err := os.Create(p.File + ".tmp")
	if err != nil {
		return err
	}

	// fields := make(map[uint]facet.KeyField)

	// for _, fld := range idx.KeyFacets {
	// 	fields[fld.Id] = *fld
	// }
	defer runtime.GC()
	defer file.Close()
	zipWriter := gzip.NewWriter(file)
	enc := gob.NewEncoder(zipWriter)
	defer zipWriter.Close()
	idx.Lock()
	defer idx.Unlock()

	for _, item := range idx.Items {
		store := item.(index.DataItem)
		err = enc.Encode(store)
		if err != nil {
			return err
		}
	}

	enc = nil
	err = os.Rename(p.File+".tmp", p.File)

	return err
}

// func toSlice(fields map[uint]interface{}) []Field {
// 	slice := make([]Field, len(fields))
// 	i := 0
// 	for id, value := range fields {
// 		slice[i] = Field{
// 			Id:    id,
// 			Value: value,
// 		}
// 		i++
// 	}
// 	return slice
// }

// func toMap(fields []Field) *facet.ItemFields {
// 	m := make(facet.ItemFields, len(fields))
// 	for _, f := range fields {
// 		m[f.Id] = f.Value
// 	}
// 	return &m
// }
