package storage

import (
	"compress/gzip"
	"encoding/gob"
	"errors"
	"io"
	"iter"
	"log"
	"os"
	"runtime"

	"github.com/bytedance/sonic"
	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/types"
)

func init() {
	gob.Register(index.DataItem{})
	gob.Register([]string{})
	gob.Register(types.ItemFields{})
	gob.Register(types.Embeddings{})
	gob.Register(map[uint]types.Embeddings{})
}

type Field struct {
	Id    uint
	Value interface{}
}

// func decodeNormal(enc *gob.Decoder, item *index.DataItem) error {

// 	err := enc.Decode(item)
// 	if err != nil {
// 		return err
// 	}
// 	// if item.AdvertisingText != "" {
// 	// 	item.Fields[21] = item.AdvertisingText
// 	// } else {
// 	// 	delete(item.Fields, 21)
// 	// }

// 	return nil
// }

func asSeq(items []types.Item) iter.Seq[types.Item] {
	return func(yield func(types.Item) bool) {
		for _, item := range items {
			if !yield(item) {
				return
			}
		}
	}
}

const itemsFile = "items.jz"
const settingsFile = "settings.json"
const facetsFile = "facets.json"
const embeddingsFile = "embeddings.gob.gz"

func (d *DiskStorage) LoadSettings() error {
	return d.LoadJson(&types.CurrentSettings, settingsFile)
}

func (d *DiskStorage) SaveSettings() error {
	types.CurrentSettings.RLock()
	defer types.CurrentSettings.RUnlock()
	return d.SaveJson(&types.CurrentSettings, settingsFile)
}

func (d *DiskStorage) LoadFacets(output interface{}) error {
	return d.LoadJson(output, facetsFile)
}

func (d *DiskStorage) SaveFacets(facets interface{}) error {
	return d.SaveJson(facets, facetsFile)
}

func (d *DiskStorage) LoadEmbeddings(output interface{}) error {
	return d.LoadGzippedGob(output, embeddingsFile)
}

func (d *DiskStorage) SaveEmbeddings(embeddings interface{}) error {
	return d.SaveGzippedGob(embeddings, embeddingsFile)
}

func (d *DiskStorage) LoadItems(handlers ...types.ItemHandler) error {
	// idx.Lock()
	// defer idx.Unlock()
	// err := p.LoadFacets(idx)
	// if err != nil {
	// 	return err
	// }

	// // Load embeddings if available
	// if err := p.LoadEmbeddings(idx); err != nil {
	// 	log.Printf("Error loading embeddings: %v", err)
	// 	// Continue loading even if embeddings failed to load
	// }
	fileName, _ := d.GetFileName(itemsFile)
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer runtime.GC()
	defer file.Close()

	zipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}

	decoder := sonic.ConfigStd.NewDecoder(zipReader)
	defer zipReader.Close()

	tmp := &index.DataItem{}
	items := make([]types.Item, 0)
	for err == nil {

		if err = decoder.Decode(tmp); err == nil {
			if tmp.IsDeleted() && !tmp.IsSoftDeleted() {
				continue
			}
			cgm, ok := tmp.Fields[35]
			if ok {
				cgmString, isString := cgm.(string)
				if isString {
					tmp.Fields[37] = cgmString[:3]
				}
			}
			items = append(items, tmp)

			//idx.UpsertItemUnsafe(tmp)
			//tmp = nil
			tmp = &index.DataItem{}
		}
	}
	for _, hs := range handlers {
		go hs.HandleItems(asSeq(items))
	}
	decoder = nil

	if errors.Is(err, io.EOF) {
		return nil
	}

	return err
}

func (p *DiskStorage) SaveGzippedJson(data any, filename string) error {
	fileName, tmpFileName := p.GetFileName(filename)

	file, err := os.Create(tmpFileName)
	if err != nil {
		return err
	}

	defer file.Close()
	defer runtime.GC()
	zipWriter := gzip.NewWriter(file)
	enc := sonic.ConfigDefault.NewEncoder(zipWriter)
	defer zipWriter.Close()

	err = enc.Encode(data)
	if err != nil {
		return err
	}

	enc = nil
	err = os.Rename(tmpFileName, fileName)
	//log.Printf("Saved file: %s", filename)

	return err
}

func (p *DiskStorage) LoadGzippedJson(data interface{}, filename string) error {
	name, _ := p.GetFileName(filename)
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()
	defer runtime.GC()

	zipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}

	enc := sonic.ConfigDefault.NewDecoder(zipReader)
	defer zipReader.Close()

	err = enc.Decode(data)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	enc = nil

	return nil
}

func (p *DiskStorage) SaveJson(data any, filename string) error {
	fileName, tmpFileName := p.GetFileName(filename)

	file, err := os.Create(tmpFileName)
	if err != nil {
		return err
	}

	defer file.Close()
	defer runtime.GC()
	enc := sonic.ConfigDefault.NewEncoder(file)

	err = enc.Encode(data)
	if err != nil {
		return err
	}

	enc = nil
	err = os.Rename(tmpFileName, fileName)
	//log.Printf("Saved file: %s", filename)

	return err
}

func (p *DiskStorage) LoadJson(data interface{}, filename string) error {
	name, _ := p.GetFileName(filename)
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()
	defer runtime.GC()

	enc := sonic.ConfigDefault.NewDecoder(file)
	defer file.Close()

	err = enc.Decode(data)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	enc = nil

	return nil
}

func (p *DiskStorage) SaveItems(items iter.Seq[types.Item]) error {
	fileName, tmpFileName := p.GetFileName(itemsFile)

	file, err := os.Create(tmpFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := gzip.NewWriter(file)
	enc := sonic.ConfigDefault.NewEncoder(zipWriter)
	defer zipWriter.Close()
	for item := range items {
		err = enc.Encode(item)
		if err != nil {
			return err
		}
	}
	enc = nil
	err = os.Rename(tmpFileName, fileName)
	if err != nil {
		log.Printf("Error renaming file: %v", err)
	}
	return nil
}

func (p *DiskStorage) SaveGzippedGob(embeddings any, name string) error {
	fileName, tmpFileName := p.GetFileName(name)

	file, err := os.Create(tmpFileName)
	if err != nil {
		return err
	}

	//defer runtime.GC()
	defer file.Close()
	zipWriter := gzip.NewWriter(file)
	enc := gob.NewEncoder(zipWriter)
	defer zipWriter.Close()

	err = enc.Encode(embeddings)
	if err != nil {
		log.Printf("Error encoding embeddings: %v", err)
		return err
	}

	err = os.Rename(tmpFileName, fileName)
	if err != nil {
		return err
	}

	return nil
}

func (p *DiskStorage) LoadGzippedGob(output interface{}, name string) error {
	fileName, _ := p.GetFileName(name)
	file, err := os.Open(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("File not found: %s", fileName)
			return nil
		}
		return err
	}

	defer runtime.GC()
	defer file.Close()

	zipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	dec := gob.NewDecoder(zipReader)

	// Decode directly into the provided output (which should be a pointer)
	err = dec.Decode(output)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	dec = nil
	return nil
}
