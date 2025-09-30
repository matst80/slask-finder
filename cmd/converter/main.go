package main

import (
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
	mu    sync.Mutex
	items []index.DataItem
}

func (c *ConverterHandler) HandleItem(item types.Item, wg *sync.WaitGroup) {
	wg.Go(func() {
		if di, ok := item.(*index.DataItem); ok {
			c.mu.Lock()
			c.items = append(c.items, *di)
			c.mu.Unlock()
		} else {
			log.Printf("Could not convert item to DataItem: %T", item)
		}

	})

}

func main() {
	s := storage.NewDiskStorage(country, "data")
	wg := sync.WaitGroup{}

	c := &ConverterHandler{
		items: make([]index.DataItem, 0),
	}

	err := s.LoadItems(&wg, c)
	if err != nil {
		log.Fatalf("Could not load items: %v", err)
	}
	wg.Wait()
	log.Printf("Done converting items, saving %d items", len(c.items))
	err = s.SaveRawItems(func(yield func(*index.RawDataItem) bool) {
		for _, item := range c.items {
			if !yield(index.NewRawConverted(&item)) { // use index to avoid pointer to loop variable copy
				return
			}
		}
	})
	if err != nil {
		log.Fatalf("Could not save items: %v", err)
	}
	log.Printf("Saved %d items", len(c.items))

}
