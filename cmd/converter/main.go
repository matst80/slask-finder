package main

import (
	"iter"
	"log"
	"os"
	"sync"

	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/storage"
	"github.com/matst80/slask-finder/pkg/types"
)

var country = "se"

func init() {
	c, ok := os.LookupEnv("COUNTRY")
	if ok {
		country = c
	}
}

type ConverterHandler struct {
	storage *storage.DiskStorage
	wg      *sync.WaitGroup
	items   []index.DataItem
}

func (c *ConverterHandler) HandleItems(itemIter iter.Seq[types.Item]) {
	defer c.wg.Done()
	for item := range itemIter {
		if di, ok := item.(*index.DataItem); ok {
			c.items = append(c.items, *di)
		} else {
			log.Printf("Could not convert item to DataItem: %T", item)
		}
	}
	err := c.storage.SaveStorageItems(func(yield func(index.StorageDataItem) bool) {
		for _, item := range c.items {
			if !yield(index.ToStorageDataItem(&item)) { // use index to avoid pointer to loop variable copy
				return
			}
		}
	})
	if err != nil {
		log.Fatalf("Could not save items: %v", err)
	}
	log.Printf("Saved %d items", len(c.items))
}

func main() {
	s := storage.NewDiskStorage(country, "data")
	wg := sync.WaitGroup{}
	wg.Add(1)
	c := &ConverterHandler{
		storage: s,
		wg:      &wg,
		items:   make([]index.DataItem, 0),
	}

	err := s.LoadItems(c)
	if err != nil {
		log.Fatalf("Could not load items: %v", err)
	}
	wg.Wait()
	log.Printf("Done converting items")
}
