package storage

import (
	"compress/gzip"
	"encoding/gob"
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

	if err.Error() == "EOF" {
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
	if err != nil {
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
	if err != nil {
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

// func (p *DiskStorage) SaveIndex(idx *index.ItemIndex) error {

// 	file, err := os.Create(p.File + ".tmp")
// 	if err != nil {
// 		return err
// 	}

// 	defer runtime.GC()
// 	defer file.Close()
// 	zipWriter := gzip.NewWriter(file)
// 	enc := sonic.ConfigDefault.NewEncoder(zipWriter)
// 	defer zipWriter.Close()

// 	for item := range idx.GetAllItems() {
// 		store, ok := item.(*index.DataItem)
// 		if !ok {
// 			log.Fatalf("Could not convert item to DataItem")
// 		}
// 		err = enc.Encode(store)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	enc = nil
// 	err = os.Rename(p.File+".tmp", p.File)

// 	if err != nil {
// 		return err
// 	}
// 	log.Println("Saved index")
// 	return nil //p.SaveFacets(idx.Facets)
// }

// func (p *DiskStorage) SaveSettings() error {
// 	types.CurrentSettings.RLock()
// 	defer types.CurrentSettings.RUnlock()
// 	return p.SaveJsonFile(types.CurrentSettings, "settings.json")
// }

// func (p *DiskStorage) LoadSettings() error {
// 	types.CurrentSettings.Lock()
// 	defer types.CurrentSettings.Unlock()
// 	return p.LoadJsonFile(types.CurrentSettings, "settings.json")
// }

// func (p *DiskStorage) SaveFacets(facets map[uint]types.Facet) error {
// 	file, err := os.Create("data/facets.json.tmp")
// 	toStore := make([]StorageFacet, 0)
// 	if err != nil {
// 		return err
// 	}
// 	defer file.Close()
// 	var base *types.BaseField
// 	for _, ff := range facets {
// 		base = ff.GetBaseField()
// 		if base != nil {
// 			toStore = append(toStore, StorageFacet{
// 				BaseField: base,
// 				Type:      FieldType(ff.GetType()),
// 			})
// 		}
// 	}
// 	err = sonic.ConfigDefault.NewEncoder(file).Encode(toStore)
// 	if err != nil {
// 		return err
// 	}
// 	return os.Rename("data/facets.json.tmp", "data/facets.json")

// }

// func LoadFacets(idx *facet.FacetItemHandler) error {
// 	file, err := os.Open("data/facets.json")
// 	if err != nil {
// 		return err
// 	}
// 	defer file.Close()
// 	toStore := make([]StorageFacet, 0)
// 	if err = sonic.ConfigDefault.NewDecoder(file).Decode(&toStore); err != nil {
// 		return err
// 	}

// 	for _, ff := range toStore {
// 		//ff.BaseField.Searchable = true
// 		if ff.BaseField.Type == "fps" {
// 			ff.BaseField.HideFacet = true
// 		}
// 		switch ff.Type {
// 		case 1:
// 			idx.AddKeyField(ff.BaseField)
// 		case 3:

// 			idx.AddIntegerField(ff.BaseField)
// 		case 2:
// 			idx.AddDecimalField(ff.BaseField)
// 		default:
// 			log.Printf("Unknown field type %d", ff.Type)
// 		}
// 	}

// 	return nil
// }

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

	err = dec.Decode(&output)
	if err != nil {
		return err
	}

	dec = nil
	return nil
}
