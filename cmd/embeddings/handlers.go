package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/matst80/slask-finder/pkg/embeddings"
	"github.com/matst80/slask-finder/pkg/storage"
	"github.com/matst80/slask-finder/pkg/types"
)

type app struct {
	country  string
	storage  *storage.DiskStorage
	index    *embeddings.ItemEmbeddingsHandler
	proxyUrl string
}

func (ws *app) CosineSimilar(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")
	id, err := strconv.ParseUint(idString, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	iid := uint(id)
	item, ok := ws.index.GetEmbeddings(iid)
	if !ok {
		http.Error(w, fmt.Sprintf("item not found with id: %d", id), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/jsonl+json; charset=UTF-8")
	ids, _ := types.FindTopSimilarEmbeddings(item, ws.index.GetAllEmbeddings(), 30)
	ws.proxyIdsToStream(w, r, ids)
}

func (ws *app) SearchEmbeddings(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	query = strings.TrimSpace(query)

	if query == "" {
		//defaultHeaders(w, r, true, "1200")
		http.Error(w, "query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	start := time.Now()
	// Generate embeddings for the query
	queryEmbeddings, err := ws.index.GetEmbeddingsEngine().GenerateEmbeddings(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to generate embeddings: %v", err), http.StatusInternalServerError)
		return
	}
	embeddingsDuration := time.Since(start)

	// Find items with similar embeddings
	start = time.Now()
	ids, _ := types.FindTopSimilarEmbeddings(queryEmbeddings, ws.index.GetAllEmbeddings(), 60)

	matchDuration := time.Since(start)
	//defaultHeaders(w, r, true, "120")
	w.Header().Set("Content-Type", "application/jsonl+json; charset=UTF-8")
	w.Header().Set("x-embeddings-duration", fmt.Sprintf("%v", embeddingsDuration))
	w.Header().Set("x-match-duration", fmt.Sprintf("%v", matchDuration))

	ws.proxyIdsToStream(w, r, ids)
}

func (ws *app) proxyIdsToStream(w http.ResponseWriter, _ *http.Request, ids []uint) {
	if len(ids) == 0 {
		w.WriteHeader(http.StatusOK)
		//w.Write([]byte("[]"))
		return
	}

	var bodyBuilder strings.Builder
	for _, id := range ids {
		fmt.Fprintln(&bodyBuilder, id)
	}

	proxyReq, err := http.NewRequest("POST", ws.proxyUrl+"/api/stream-items", strings.NewReader(bodyBuilder.String()))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create proxy request: %v", err), http.StatusInternalServerError)
		return
	}
	proxyReq.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	proxyResp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to execute proxy request: %v", err), http.StatusInternalServerError)
		return
	}
	defer proxyResp.Body.Close()

	w.WriteHeader(proxyResp.StatusCode)

	if _, err := io.Copy(w, proxyResp.Body); err != nil {
		log.Printf("failed to copy response body: %v", err)
	}
}
