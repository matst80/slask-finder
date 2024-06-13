package persistance

import (
	"encoding/gob"
	"io"
	"os"

	"tornberg.me/facet-search/pkg/search"
)

func (p *Persistance) LoadFreeText(ft *search.FreeTextIndex) error {
	file, err := os.Open(p.FreeTextFile)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := io.Reader(file)
	enc := gob.NewDecoder(reader)

	err = enc.Decode(ft)
	if err != nil {
		return err
	}

	enc = nil
	return nil
}

func (p *Persistance) SaveFreeText(ft *search.FreeTextIndex) error {
	file, err := os.Create(p.FreeTextFile)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := io.Writer(file)
	enc := gob.NewEncoder(writer)
	err = enc.Encode(ft)
	if err != nil {
		return err
	}

	return nil
}
