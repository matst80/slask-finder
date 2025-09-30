package embeddings_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/matst80/slask-finder/pkg/embeddings"
)

// TestMultipleEndpointsLoadTesting demonstrates using multiple endpoints with concurrent requests
func TestMultipleEndpointsLoadTesting(t *testing.T) {
	// Skip this test in normal test runs as it requires actual Ollama servers
	t.Skip("Skipping load test which requires actual Ollama servers")

	// Create engine with multiple Ollama endpoints
	engine := embeddings.NewOllamaEmbeddingsEngineWithMultipleEndpoints(
		"nomic-embed-text",

		"http://server1:11434/api/embeddings",
		"http://server2:11434/api/embeddings",
		"http://server3:11434/api/embeddings",
	)

	// Number of concurrent requests to simulate
	concurrency := 20
	requestsPerWorker := 5
	totalRequests := concurrency * requestsPerWorker

	// Track metrics
	var (
		wg           sync.WaitGroup
		mu           sync.Mutex
		errorCount   int
		successCount int
	)

	// Start timing
	startTime := time.Now()

	// Launch concurrent workers
	for i := range concurrency {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := range requestsPerWorker {
				// Create a unique text for each request
				text := fmt.Sprintf("Sample text for embeddings generation - worker %d, request %d", workerID, j)

				// Generate embeddings
				_, err := engine.GenerateEmbeddings(text)

				// Track result
				mu.Lock()
				if err != nil {
					errorCount++
					t.Logf("Error in worker %d, request %d: %v", workerID, j, err)
				} else {
					successCount++
				}
				mu.Unlock()

				// Small delay between requests from the same worker
				time.Sleep(50 * time.Millisecond)
			}
		}(i)
	}

	// Wait for all workers to complete
	wg.Wait()
	duration := time.Since(startTime)

	// Report results
	t.Logf("Load test results:")
	t.Logf("- Total requests: %d", totalRequests)
	t.Logf("- Successful: %d (%.1f%%)", successCount, float64(successCount)/float64(totalRequests)*100)
	t.Logf("- Failed: %d (%.1f%%)", errorCount, float64(errorCount)/float64(totalRequests)*100)
	t.Logf("- Total duration: %v", duration)
	t.Logf("- Requests per second: %.1f", float64(totalRequests)/duration.Seconds())

	// Verify all requests were processed
	if successCount+errorCount != totalRequests {
		t.Errorf("Expected %d total processed requests, got %d", totalRequests, successCount+errorCount)
	}

	// Fail the test if there were any errors
	if errorCount > 0 {
		t.Errorf("Expected 0 errors, got %d", errorCount)
	}
}
