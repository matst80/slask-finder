package storage

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

func NewPersistance() *DataRepository {
	gob.Register(index.DataItem{})
	gob.Register([]string{})
	gob.Register(types.ItemFields{})
	// gob.Register([]interface{}(nil))
	return &DataRepository{
		File:         "data/index-v2.jz",
		FreeTextFile: "data/freetext.dbz",
	}
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

func (p *DataRepository) LoadIndex(idx *index.Index) error {
	idx.Lock()
	defer idx.Unlock()
	err := p.LoadFacets(idx)
	if err != nil {
		return err
	}
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

func (p *DataRepository) SaveJsonFile(data interface{}, filename string) error {
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
	log.Printf("Saved file: %s", filename)

	return err
}

func (p *DataRepository) LoadJsonFile(data interface{}, filename string) error {
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

func (p *DataRepository) SaveIndex(idx *index.Index) error {

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
		store, ok := item.(*index.DataItem)
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

	go p.SaveSettings()

	if err != nil {
		return err
	}
	log.Println("Saved index")
	return p.SaveFacets(idx.Facets)
}

func (p *DataRepository) SaveSettings() error {
	types.CurrentSettings.RLock()
	defer types.CurrentSettings.RUnlock()
	return p.SaveJsonFile(types.CurrentSettings, "settings.json")
}

func (p *DataRepository) LoadSettings() error {
	types.CurrentSettings.Lock()
	defer types.CurrentSettings.Unlock()
	return p.LoadJsonFile(types.CurrentSettings, "settings.json")
}

type FieldType uint

type StorageFacet struct {
	*types.BaseField
	Type FieldType `json:"type"`
}

func (p *DataRepository) SaveFacets(facets map[uint]types.Facet) error {
	file, err := os.Create("data/facets.json.tmp")
	toStore := make([]StorageFacet, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	var base *types.BaseField
	for _, ff := range facets {
		base = ff.GetBaseField()
		if base != nil {
			toStore = append(toStore, StorageFacet{
				BaseField: base,
				Type:      FieldType(ff.GetType()),
			})
		}
	}
	err = json.NewEncoder(file).Encode(toStore)
	if err != nil {
		return err
	}
	return os.Rename("data/facets.json.tmp", "data/facets.json")

}

func (p *DataRepository) LoadFacets(idx *index.Index) error {
	file, err := os.Open("data/facets.json")
	if err != nil {
		return err
	}
	defer file.Close()
	toStore := make([]StorageFacet, 0)
	if err = json.NewDecoder(file).Decode(&toStore); err != nil {
		return err
	}

	for _, ff := range toStore {
		//ff.BaseField.Searchable = true
		if ff.BaseField.Type == "fps" {
			ff.BaseField.HideFacet = true
		}
		switch ff.Type {
		case 1:
			if ff.BaseField.LinkedId != 0 {
				log.Printf("Linked field %d -> %d", ff.BaseField.Id, ff.BaseField.LinkedId)
			}
			idx.AddKeyField(ff.BaseField)
		case 3:

			idx.AddIntegerField(ff.BaseField)
		case 2:
			idx.AddDecimalField(ff.BaseField)
		default:
			log.Printf("Unknown field type %d", ff.Type)
		}
	}

	return nil
}
