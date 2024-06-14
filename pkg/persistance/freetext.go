package persistance

import (
	"compress/gzip"
	"encoding/gob"
	"os"

	"tornberg.me/facet-search/pkg/search"
)

func (p *Persistance) LoadFreeText(ft *search.FreeTextIndex) error {
	file, err := os.Open(p.FreeTextFile)
	if err != nil {
		return err
	}

	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	enc := gob.NewDecoder(reader)
	for err == nil {
		var doc search.Document
		err = enc.Decode(&doc)
		if err == nil {
			ft.AddDocument(&doc)
		}
	}
	if err.Error() == "EOF" {
		return nil
	}
	return err
}

func (p *Persistance) SaveFreeText(ft *search.FreeTextIndex) error {
	file, err := os.Create(p.FreeTextFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := gzip.NewWriter(file)

	enc := gob.NewEncoder(writer)
	for _, doc := range ft.Documents {
		err = enc.Encode(*doc)
		if err != nil {
			return err
		}
	}
	writer.Close()

	return nil
}
