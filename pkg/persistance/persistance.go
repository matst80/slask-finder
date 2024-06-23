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
	var tmp = &index.StorageItem{}
	for err == nil {

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

func getStorageFields(input facet.ItemFields) index.DataItemFields {

	fields := make([]index.KeyFieldValue, len(input.Fields))
	for id, value := range input.Fields {
		fields = append(fields, index.KeyFieldValue{
			Id:    id,
			Value: value,
		})

	}
	decimalFields := make([]index.DecimalFieldValue, 0)
	for id, value := range input.DecimalFields {
		decimalFields = append(decimalFields, index.DecimalFieldValue{
			Id:    id,
			Value: value,
		})
	}
	integerFields := make([]index.IntegerFieldValue, 0)
	for id, value := range input.IntegerFields {
		integerFields = append(integerFields, index.IntegerFieldValue{
			Id:    id,
			Value: value,
		})
	}

	return index.DataItemFields{
		Fields:        fields,
		DecimalFields: decimalFields,
		IntegerFields: integerFields,
	}
}

func convertStorageItem(item *index.DataItem, output *index.StorageItem) {
	output.BaseItem = item.BaseItem
	output.DataItemFields = getStorageFields(item.ItemFields)
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
	var storageItem = &index.StorageItem{}
	for _, item := range idx.Items {
		convertStorageItem(item, storageItem)
		err = enc.Encode(storageItem)
		if err != nil {
			return err
		}
	}

	enc = nil

	return nil
}
