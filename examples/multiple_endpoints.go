package main

import (
	"fmt"
	"sync"

	"github.com/matst80/slask-finder/pkg/embeddings"
	"github.com/matst80/slask-finder/pkg/index"
	"github.com/matst80/slask-finder/pkg/types"
)

func main() {
	// Create engine with multiple Ollama endpoints for round-robin load balancing
	engine := embeddings.NewOllamaEmbeddingsEngineWithMultipleEndpoints(
		"nomic-embed-text", // Model to use

		"http://server1:11434/api/embeddings",
		"http://server2:11434/api/embeddings",
		"http://server3:11434/api/embeddings",
	)

	// Initialize index with the embeddings engine
	idx := index.NewIndex(engine, func(i *index.Index) error { return nil })

	// Example: Generate embeddings for multiple items concurrently
	// This will automatically distribute requests across all endpoints
	var wg sync.WaitGroup
	items := []string{
		"Smartphone with high-resolution camera",
		"Leather wallet with multiple card slots",
		"Wireless noise-cancelling headphones",
		"Stainless steel water bottle",
		"Ergonomic office chair with lumbar support",
		"Fitness tracker with heart rate monitor",
		"Portable bluetooth speaker",
		"Mechanical keyboard with RGB lighting",
		"Ultrawide curved monitor",
		"Electric standing desk",
	}

	// Process multiple items concurrently
	for i, text := range items {
		wg.Add(1)
		go func(id int, itemText string) {
			defer wg.Done()

			// Create a mock item
			item := &types.MockItem{
				Id:     uint(id),
				Title:  itemText,
				Fields: make(map[uint]interface{}),
			}

			// Add item to index (this will generate embeddings using round-robin)
			idx.UpsertItem(item)

			fmt.Printf("Added item %d: %s\n", id, itemText)
		}(i+1, text)
	}

	// Wait for all items to be processed
	wg.Wait()
	fmt.Println("All items processed using round-robin across multiple Ollama endpoints")
}
