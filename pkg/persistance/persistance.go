package persistance

import (
	"compress/gzip"
	"encoding/gob"
	"log"
	"os"
	"runtime"

	"github.com/matst80/slask-finder/pkg/index"
)

func NewPersistance() *Persistance {
	gob.Register(index.DataItem{})
	// gob.Register([]interface{}(nil))
	return &Persistance{
		File:         "data/index-v2.dbz",
		FreeTextFile: "data/freetext.dbz",
	}
}

type Field struct {
	Id    uint
	Value interface{}
}

func decodeNormal(enc *gob.Decoder, item *index.DataItem) error {

	err := enc.Decode(item)
	return err
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

		if err = decodeOld(enc, tmp); err == nil {
			idx.UpsertItemUnsafe(tmp)
			tmp = &index.DataItem{}
		}
	}
	enc = nil

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
		store, ok := (*item).(*index.DataItem)
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
