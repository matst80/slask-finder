package persistance

import (
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"log"
	"os"
	"path"
	"runtime"

	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/types"
)

func NewPersistance() *Persistance {
	gob.Register(index.DataItem{})
	gob.Register([]string{})
	gob.Register(types.ItemFields{})
	// gob.Register([]interface{}(nil))
	return &Persistance{
		File:         "data/index-v2.jz",
		FreeTextFile: "data/freetext.dbz",
	}
}

type Field struct {
	Id    uint
	Value interface{}
}

func decodeNormal(enc *gob.Decoder, item *index.DataItem) error {

	err := enc.Decode(item)
	if err != nil {
		return err
	}
	// if item.AdvertisingText != "" {
	// 	item.Fields[21] = item.AdvertisingText
	// } else {
	// 	delete(item.Fields, 21)
	// }

	return nil
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

	enc := json.NewDecoder(zipReader)
	defer zipReader.Close()

	idx.Lock()
	defer idx.Unlock()
	tmp := &index.DataItem{}
	for err == nil {

		if err = enc.Decode(tmp); err == nil {
			if tmp.IsDeleted() && !tmp.IsSoftDeleted() {
				continue
			}
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

func (p *Persistance) SaveJsonFile(data interface{}, filename string) error {
	tmpFileName := path.Join("data", filename+".tmp")
	file, err := os.Create(tmpFileName)
	if err != nil {
		return err
	}

	defer runtime.GC()
	defer file.Close()
	zipWriter := gzip.NewWriter(file)
	enc := json.NewEncoder(zipWriter)
	defer zipWriter.Close()

	err = enc.Encode(data)
	if err != nil {
		return err
	}

	enc = nil
	err = os.Rename(tmpFileName, path.Join("data", filename))
	log.Println("Saved index")

	return err
}

func (p *Persistance) LoadJsonFile(data interface{}, filename string) error {
	file, err := os.Open(path.Join("data", filename))
	if err != nil {
		return err
	}
	defer runtime.GC()
	defer file.Close()

	zipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}

	enc := json.NewDecoder(zipReader)
	defer zipReader.Close()

	err = enc.Decode(data)
	if err != nil {
		return err
	}

	enc = nil

	return nil
}

func (p *Persistance) SaveIndex(idx *index.Index) error {

	file, err := os.Create(p.File + ".tmp")
	if err != nil {
		return err
	}

	defer runtime.GC()
	defer file.Close()
	zipWriter := gzip.NewWriter(file)
	enc := json.NewEncoder(zipWriter)
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
	log.Println("Saved index")

	return err
}
