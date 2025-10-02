package storage

import (
	"fmt"
	"path"
	"time"
)

type DiskStorage struct {
	Country    string
	RootFolder string
}

func NewDiskStorage(country, rootFolder string) *DiskStorage {
	return &DiskStorage{
		Country:    country,
		RootFolder: rootFolder,
	}
}

func (ds *DiskStorage) GetFileName(name string) (string, string) {
	fileName := path.Join(ds.RootFolder, ds.Country, name)
	tmpFileName := fileName + ".tmp-" + fmt.Sprintf("%d", time.Now().UnixMilli())
	return fileName, tmpFileName
}

func (ds *DiskStorage) GetOverrideFilename(name string) string {
	fileName := path.Join(ds.RootFolder, "overrides", name)
	return fileName
}
