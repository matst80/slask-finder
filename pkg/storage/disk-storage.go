package storage

import (
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"errors"
	"io"
	"iter"
	"log"
	"os"
	"runtime"
	"slices"
	"sync"

	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/types"
)

func init() {
	gob.Register(index.RawDataItem{})
	gob.Register(index.DataItem{})
	gob.Register([]string{})
	gob.Register(types.ItemFields{})
	gob.Register(types.Embeddings{})
	gob.Register(map[uint]types.Embeddings{})
}

type Field struct {
	Id    uint
	Value any
}

const itemsFile = "items.jz"
const storageItemFile = "items-v3.gz"
const settingsFile = "settings.json"
const legacySettingsFile = "settings.jz"
const facetsFile = "facets.json"
const embeddingsFile = "embeddings.gob.gz"

func (d *DiskStorage) LoadSettings() error {
	types.CurrentSettings.Lock()
	defer types.CurrentSettings.Unlock()
	legacyPath, _ := d.GetFileName(legacySettingsFile)
	// Try loading legacy gzipped settings first
	f, err := os.Stat(legacyPath)
	if err == nil && !f.IsDir() {
		log.Printf("Loading legacy settings file: %s", legacyPath)
		if err = d.LoadGzippedJson(&types.CurrentSettings, legacySettingsFile); err == nil {
			log.Printf("Successfully loaded legacy settings, saving to new format")
			if err = d.SaveJson(&types.CurrentSettings, settingsFile); err != nil {
				log.Printf("Failed to save new settings format: %v", err)
			} else {
				if removeErr := os.Remove(legacyPath); removeErr != nil && !os.IsNotExist(removeErr) {
					log.Printf("Failed to remove legacy settings file: %v", removeErr)
				}
				if removeErr := os.Remove(legacyPath + ".bak"); removeErr != nil && !os.IsNotExist(removeErr) {
					log.Printf("Failed to remove legacy settings backup file: %v", removeErr)
				}
			}
			return nil
		}
		log.Printf("Failed to load legacy settings: %v", err)

	}
	return d.LoadJson(&types.CurrentSettings, settingsFile)
}

func (d *DiskStorage) SaveSettings() error {
	types.CurrentSettings.RLock()
	defer types.CurrentSettings.RUnlock()
	return d.SaveJson(&types.CurrentSettings, settingsFile)
}

func (d *DiskStorage) LoadFacets(output *[]types.StorageFacet) error {
	return d.LoadJson(output, facetsFile)
}

func (d *DiskStorage) SaveFacets(facets *[]types.StorageFacet) error {
	return d.SaveJson(facets, facetsFile)
}

func (d *DiskStorage) LoadEmbeddings(output *map[uint]types.Embeddings) error {
	return d.LoadGzippedGob(output, embeddingsFile)
}

func (d *DiskStorage) SaveEmbeddings(embeddings *map[uint]types.Embeddings) error {
	return d.SaveGzippedGob(embeddings, embeddingsFile)
}

func (d *DiskStorage) StreamContent(w io.Writer, fileName string) (int64, error) {
	osFileName, _ := d.GetFileName(fileName)
	file, err := os.Open(osFileName)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	return file.WriteTo(w)

}

// func (d *DiskStorage) loadNewItems(fileName string, handlers ...types.ItemHandler) error {
// 	file, err := os.Open(fileName)
// 	if err != nil {
// 		return err
// 	}
// 	defer runtime.GC()
// 	defer file.Close()

// 	zipReader, err := gzip.NewReader(file)
// 	if err != nil {
// 		return err
// 	}
// 	defer zipReader.Close()

// 	decoder := gob.NewDecoder(zipReader)
// 	defer zipReader.Close()

// 	tmp := make([]*index.RawDataItem, 0)

// 	err = decoder.Decode(&tmp)
// 	log.Printf("Loaded %d items from %s", len(tmp), fileName)
// 	for _, hs := range handlers {
// 		go hs.HandleItems(asSeq(tmp))
// 	}
// 	decoder = nil
// 	tmp = nil

// 	if errors.Is(err, io.EOF) {
// 		return nil
// 	}

//		return err
//	}
func (d *DiskStorage) LoadSortOverride(name string) (*types.SortOverride, error) {
	fileName := d.GetOverrideFilename(name)

	b, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	tmp := &types.SortOverride{}
	err = tmp.FromString(string(b))
	return tmp, err
}

func (d *DiskStorage) LoadItems(wg *sync.WaitGroup, handlers ...types.ItemHandler) error {
	// newFileName, _ := d.GetFileName(storageItemFile)
	// _, err := os.Stat(newFileName)
	// if err == nil {
	// 	return d.loadNewItems(newFileName, handlers...)
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

	decoder := json.NewDecoder(zipReader)
	defer zipReader.Close()

	tmp := &index.DataItem{}
	//items := make([]types.Item, 0)
	for err == nil {

		if err = decoder.Decode(tmp); err == nil {
			if tmp.IsDeleted() && !tmp.IsSoftDeleted() {
				continue
			}
			// cgm, ok := tmp.Fields[35]
			// if ok {
			// 	cgmString, isString := cgm.(string)
			// 	if isString {
			// 		tmp.Fields[37] = cgmString[:3]
			// 	}
			// }
			for _, hs := range handlers {
				go hs.HandleItem(tmp, wg)
			}
			//items = append(items, tmp)

			tmp = &index.DataItem{}
		}
	}
	// for _, hs := range handlers {
	// 	go hs.HandleItems(slices.Values(items))
	// }
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
	enc := json.NewEncoder(zipWriter)
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

func (p *DiskStorage) LoadGzippedJson(data any, filename string) error {
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

	enc := json.NewDecoder(zipReader)
	defer zipReader.Close()

	err = enc.Decode(data)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	return nil
}

func (p *DiskStorage) SaveJson(data any, name string) error {
	fileName, tmpFileName := p.GetFileName(name)

	file, err := os.Create(tmpFileName)
	if err != nil {
		return err
	}

	defer runtime.GC()
	enc := json.NewEncoder(file)

	err = enc.Encode(data)
	file.Close()
	if err != nil {
		return err
	}

	err = os.Rename(tmpFileName, fileName)
	//log.Printf("Saved file: %s", filename)

	return err
}

func (p *DiskStorage) LoadJson(data any, filename string) error {
	name, _ := p.GetFileName(filename)
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()
	defer runtime.GC()

	enc := json.NewDecoder(file)
	err = enc.Decode(data)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	enc = nil

	return nil
}

func (p *DiskStorage) SaveRawItems(items iter.Seq[*index.RawDataItem]) error {
	fileName, tmpFileName := p.GetFileName(storageItemFile)

	file, err := os.Create(tmpFileName)
	if err != nil {
		return err
	}
	defer file.Close()
	zipWriter := gzip.NewWriter(file)
	enc := gob.NewEncoder(zipWriter)
	defer zipWriter.Close()
	toStore := slices.Collect(items)
	log.Printf("Saving %d items to %s", len(toStore), fileName)
	err = enc.Encode(toStore)

	if err != nil {
		return err
	}

	err = os.Rename(tmpFileName, fileName)
	if err != nil {
		log.Printf("Error renaming file: %v", err)
	}
	return nil
}

func (p *DiskStorage) SaveItems(items iter.Seq[types.Item]) error {
	fileName, tmpFileName := p.GetFileName(itemsFile)

	file, err := os.Create(tmpFileName)
	if err != nil {
		return err
	}

	zipWriter := gzip.NewWriter(file)
	enc := json.NewEncoder(zipWriter)
	defer zipWriter.Close()
	for item := range items {
		baseItem, ok := item.(*index.DataItem)
		if !ok {
			log.Printf("Warning: item is not of type *index.DataItem, skipping, got %T", item)
			continue
		}
		err = enc.Encode(baseItem)
		if err != nil {
			break
		}
	}
	file.Close()
	if err != nil {
		return err
	}

	enc = nil
	err = os.Rename(tmpFileName, fileName)
	if err != nil {
		log.Printf("Error renaming file: %v", err)
	}
	return err
}

func (p *DiskStorage) SaveGzippedGob(embeddings any, name string) error {
	fileName, tmpFileName := p.GetFileName(name)

	file, err := os.Create(tmpFileName)
	if err != nil {
		return err
	}

	zipWriter := gzip.NewWriter(file)
	enc := gob.NewEncoder(zipWriter)

	if err = enc.Encode(embeddings); err != nil {
		log.Printf("Error encoding embeddings: %v", err)
		_ = zipWriter.Close()
		_ = file.Close()
		_ = os.Remove(tmpFileName)
		return err
	}

	if err = zipWriter.Close(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpFileName)
		return err
	}

	if err = file.Close(); err != nil {
		_ = os.Remove(tmpFileName)
		return err
	}

	if err = os.Rename(tmpFileName, fileName); err != nil {
		_ = os.Remove(tmpFileName)
		return err
	}

	return nil
}

func (p *DiskStorage) LoadGzippedGob(output any, name string) error {
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
