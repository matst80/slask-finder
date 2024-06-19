package persistance

import (
	"compress/gzip"
	"encoding/gob"
	"os"
	"runtime"

	"tornberg.me/facet-search/pkg/index"
)

func NewPersistance() *Persistance {
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}(nil))
	return &Persistance{
		File:         "data/index-v2.dbz",
		FreeTextFile: "data/freetext.dbz",
	}
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
	for err == nil {
		var tmp = &index.DataItem{}
		if err = enc.Decode(tmp); err == nil {

			idx.UpsertItemUnsafe(tmp)
		}
	}
	enc = nil
	//v = nil
	if err.Error() == "EOF" {
		return nil
	}

	return err
}

func (p *Persistance) SaveIndex(idx *index.Index) error {

	file, err := os.Create(p.File)
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

	for _, item := range idx.Items {
		err = enc.Encode(*item)
		if err != nil {
			return err
		}
	}

	enc = nil

	return nil
}
