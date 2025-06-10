package types

import (
	"math"
)

// CosineSimilarity calculates the cosine similarity between two Embeddings vectors
func CosineSimilarity(a, b Embeddings) float64 {
	// Return 0 if either vector is empty or they have different lengths
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}

	var dotProduct float64
	var normA float64
	var normB float64

	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	// Check for zero norm to avoid division by zero
	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// FindMostSimilarEmbeddings takes a query embedding and a map of ID -> Embeddings
// and returns the ID of the most similar embedding and its similarity score
func FindMostSimilarEmbeddings(query Embeddings, embeddings map[uint]Embeddings) (bestID uint, similarity float64) {
	bestSimilarity := -1.0 // Initialize with value below possible cosine similarity range

	for id, vec := range embeddings {
		sim := CosineSimilarity(query, vec)
		if sim > bestSimilarity {
			bestSimilarity = sim
			bestID = id
		}
	}

	return bestID, bestSimilarity
}

// FindTopSimilarEmbeddings takes a query embedding, a map of ID -> Embeddings, and the number of results to return
// It returns a slice of IDs sorted by similarity (highest first) along with their similarity scores
func FindTopSimilarEmbeddings(query Embeddings, embeddings map[uint]Embeddings, topN int) ([]uint, []float64) {
	type result struct {
		id         uint
		similarity float64
	}

	// Create a slice to hold all similarities
	results := make([]result, 0, len(embeddings))

	// Calculate similarities
	for id, vec := range embeddings {
		sim := CosineSimilarity(query, vec)
		results = append(results, result{id, sim})
	}

	// Sort by similarity (descending)
	resultsSorted := make([]result, len(results))
	copy(resultsSorted, results)

	for i := 0; i < len(resultsSorted); i++ {
		maxIdx := i
		for j := i + 1; j < len(resultsSorted); j++ {
			if resultsSorted[j].similarity > resultsSorted[maxIdx].similarity {
				maxIdx = j
			}
		}
		// Swap
		resultsSorted[i], resultsSorted[maxIdx] = resultsSorted[maxIdx], resultsSorted[i]
	}

	// Limit to topN results
	if topN > 0 && topN < len(resultsSorted) {
		resultsSorted = resultsSorted[:topN]
	}

	// Extract IDs and similarities
	ids := make([]uint, len(resultsSorted))
	similarities := make([]float64, len(resultsSorted))

	for i, r := range resultsSorted {
		ids[i] = r.id
		similarities[i] = r.similarity
	}

	return ids, similarities
}
