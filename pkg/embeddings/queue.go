package embeddings

import (
	"log"
	"sync"
	"time"

	"github.com/matst80/slask-finder/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	embedQueueSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "slaskfinder_embeddings_queue_size",
		Help: "The current number of items in the embeddings generation queue",
	})
	embedProcessedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "slaskfinder_embeddings_processed_total",
		Help: "The total number of embeddings generated",
	})
	embedErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "slaskfinder_embeddings_errors_total",
		Help: "The total number of embedding generation errors",
	})
)

// EmbeddingJob represents a job to generate embeddings for an item
type EmbeddingJob struct {
	Text      string
	Id        uint
	CreatedAt time.Time
	StartedAt time.Time
}

// EmbeddingsQueue manages a queue of items for embedding generation
// with a worker pool to limit concurrency
type EmbeddingsQueue struct {
	engine      types.EmbeddingsEngine
	queue       chan EmbeddingJob
	storeFunc   func(uint, types.Embeddings)
	doneFunc    func() error
	workerCount int
	wg          sync.WaitGroup
	stopCh      chan struct{}
	//mu          sync.RWMutex
}

// NewEmbeddingsQueue creates a new embeddings queue with the specified
// number of workers and buffer size
func NewEmbeddingsQueue(
	engine types.EmbeddingsEngine,
	storeFunc func(uint, types.Embeddings),
	whenQueueIsDone func() error,
	workerCount int,
	queueSize int,
) *EmbeddingsQueue {
	if workerCount <= 0 {
		workerCount = 2 // Default to 2 workers
	}
	if queueSize <= 0 {
		queueSize = 1000 // Default queue size
	}

	return &EmbeddingsQueue{
		engine:      engine,
		queue:       make(chan EmbeddingJob, queueSize),
		doneFunc:    whenQueueIsDone,
		storeFunc:   storeFunc,
		workerCount: workerCount,
		stopCh:      make(chan struct{}),
	}
}

// Start initializes and starts the worker pool
func (eq *EmbeddingsQueue) Start() {

	eq.wg.Add(eq.workerCount)

	for i := 0; i < eq.workerCount; i++ {
		go eq.worker(i)
	}

	log.Printf("Started embeddings queue with %d workers", eq.workerCount)
}

// Stop gracefully stops the worker pool
func (eq *EmbeddingsQueue) Stop() {

	close(eq.stopCh)
	eq.wg.Wait()

	log.Println("Embeddings queue stopped")
}

// QueueItem adds an item to the embeddings generation queue
// Returns true if queued successfully, false if queue is full or not running
func (eq *EmbeddingsQueue) QueueItem(item types.Item) bool {

	// Try to add to queue immediately first
	select {
	case eq.queue <- EmbeddingJob{
		Text:      buildItemRepresentation(item),
		Id:        item.GetId(),
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}:
		embedQueueSize.Inc()
		return true
	default:
		// Queue is full, but try with a timeout before giving up
		log.Printf("Embeddings queue full, waiting to add item %d...", item.GetId())
	}
	return false
}

// QueueItems adds multiple items to the embeddings generation queue
// Returns the number of successfully queued items
func (eq *EmbeddingsQueue) QueueItems(items []types.Item) int {
	if len(items) == 0 {
		return 0
	}

	successCount := 0
	for _, item := range items {
		// Try immediately with no waiting
		select {
		case eq.queue <- EmbeddingJob{
			Text:      buildItemRepresentation(item),
			Id:        item.GetId(),
			CreatedAt: time.Now(),
		}:
			embedQueueSize.Inc()
			successCount++
		default:
			// Skip this item if the queue is full
			continue
		}
	}

	if successCount < len(items) {
		log.Printf("Added %d/%d items to embeddings queue (queue full for remaining items)",
			successCount, len(items))
	}

	return successCount
}

// QueueLength returns the current number of items in the queue
func (eq *EmbeddingsQueue) QueueLength() int {
	return len(eq.queue)
}

// QueueCapacity returns the total capacity of the queue
func (eq *EmbeddingsQueue) QueueCapacity() int {
	return cap(eq.queue)
}

// Status returns the current status of the embeddings queue
func (eq *EmbeddingsQueue) Status() map[string]interface{} {

	queueLen := len(eq.queue)
	queueCap := cap(eq.queue)

	// Calculate utilization with protection against division by zero
	var utilization float64
	if queueCap > 0 {
		utilization = float64(queueLen) / float64(queueCap)
	}

	// Calculate estimated time based on worker count
	estimatedTime := estimateTimeLeft(queueLen, float64(eq.workerCount))

	return map[string]interface{}{
		"workerCount":       eq.workerCount,
		"queueLength":       queueLen,
		"queueCapacity":     queueCap,
		"utilization":       utilization,
		"utilizationPct":    utilization * 100.0,
		"estimatedTimeLeft": estimatedTime.String(),
		"estimatedSeconds":  estimatedTime.Seconds(),
		"timestamp":         time.Now().Format(time.RFC3339),
	}
}

// worker processes jobs from the queue
func (eq *EmbeddingsQueue) worker(id int) {
	defer eq.wg.Done()

	log.Printf("Embeddings worker %d started", id)

	for {
		select {
		case job, ok := <-eq.queue:
			if !ok {
				// Queue was closed
				return
			}

			embedQueueSize.Dec()

			// Process the job
			itemId := job.Id

			embeddings, err := eq.engine.GenerateEmbeddings(job.Text)
			if err != nil {
				log.Printf("Worker %d: Failed to generate embeddings for item %d: %v", id, itemId, err)
				embedErrorsTotal.Inc()
				continue
			}

			// Store the embeddings using the provided store function
			eq.storeFunc(itemId, embeddings)
			embedProcessedTotal.Inc()

			// Log processing time and remaining queue items
			processingTime := time.Since(job.CreatedAt)
			remainingItems := len(eq.queue)
			log.Printf("Worker %d: Generated embeddings for item %d in %v, remaining items in queue: %d", id, itemId, processingTime, remainingItems)

		case <-eq.stopCh:
			// Stop signal received
			log.Printf("Embeddings worker %d stopping", id)
			eq.doneFunc()
			return
		}
	}
}

// estimateTimeLeft calculates the estimated time to process remaining queue items
func estimateTimeLeft(queueLength int, workerCount float64) time.Duration {
	if queueLength == 0 {
		return 0
	}

	// Assume each worker can process approximately 1 item per second
	// This is a simplification since actual processing time depends on embeddings engine
	timeInSeconds := float64(queueLength) / workerCount

	// Convert to duration
	return time.Duration(timeInSeconds * float64(time.Second))
}
