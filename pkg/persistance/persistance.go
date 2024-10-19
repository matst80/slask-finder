package persistance

import (
	"compress/gzip"
	"encoding/gob"
	"log"
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

type Field struct {
	Id    uint
	Value interface{}
}

func decodeNormal(enc *gob.Decoder) (*index.DataItem, error) {
	tmp := &index.DataItem{}
	err := enc.Decode(tmp)
	return tmp, err
}

func decodeOld(enc *gob.Decoder) (*index.DataItem, error) {
	tmp := &StoredItem{}
	err := enc.Decode(tmp)
	if err == nil {
		fields := make(facet.ItemFields)
		for _, field := range tmp.Fields {
			fields[field.Id] = field.Value
		}
		for _, field := range tmp.DecimalFields {
			fields[field.Id] = field.Value
		}
		for _, field := range tmp.IntegerFields {
			fields[field.Id] = field.Value
		}
		return &index.DataItem{
			BaseItem: tmp.BaseItem,
			Fields:   fields,
		}, nil
	}
	return nil, err
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
	tmp := &index.DataItem{}
	for err == nil {

		if tmp, err = decodeNormal(enc); err == nil {
			idx.UpsertItemUnsafe(tmp)
		}
	}
	enc = nil

	tmp = nil
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

	defer runtime.GC()
	defer file.Close()
	zipWriter := gzip.NewWriter(file)
	enc := gob.NewEncoder(zipWriter)
	defer zipWriter.Close()
	idx.Lock()
	defer idx.Unlock()

	for _, item := range idx.Items {
		store, ok := (*item).(index.DataItem)
		if !ok {
			log.Fatalf("Could not convert item to DataItem")
		}
		err = enc.Encode(store)
		if err != nil {
			return err
		}
	}

	enc = nil
	err = os.Rename(p.File+".tmp", p.File)

	return err
}
