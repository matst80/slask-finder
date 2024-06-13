package persistance

import (
	"encoding/gob"
	"io"
	"os"
)

func NewGobPersistance() *GobPersistance {
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}(nil))
	return &GobPersistance{}
}

func (p *GobPersistance) Load(fileName string, data any) error {

	file, err := os.Open(p.File)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := io.Reader(file)
	enc := gob.NewDecoder(reader)

	err = enc.Decode(data)

	if err != nil {
		return err
	}

	return nil
}

func (p *GobPersistance) Save(fileName string, data *[]interface{}) error {

	file, err := os.Create(p.File)
	if err != nil {
		return err
	}

	defer file.Close()
	writer := io.Writer(file)
	enc := gob.NewEncoder(writer)
	err = enc.Encode(data)
	if err != nil {
		return err
	}
	enc = nil
	return nil
}
