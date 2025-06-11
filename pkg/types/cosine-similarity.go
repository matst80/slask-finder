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

	// If topN is invalid or zero, calculate for all embeddings
	if topN <= 0 || topN > len(embeddings) {
		topN = len(embeddings)
	}

	// Create a slice to hold only the top N results
	results := make([]result, 0, topN)

	// Calculate similarities and insert in sorted position
	for id, vec := range embeddings {
		sim := CosineSimilarity(query, vec)

		if len(results) < topN {
			// If we haven't filled our results slice yet, add the new result
			// in the correct sorted position
			inserted := false
			for i, r := range results {
				if sim > r.similarity {
					// Insert at position i
					results = append(results, result{}) // Make space
					copy(results[i+1:], results[i:])    // Shift elements to the right
					results[i] = result{id, sim}        // Insert new element
					inserted = true
					break
				}
			}
			if !inserted {
				// If we didn't insert in the middle, append to the end
				results = append(results, result{id, sim})
			}
		} else if sim > results[len(results)-1].similarity {
			// If our results slice is full but this similarity is higher than the lowest one
			// Find the right position to insert
			pos := len(results) - 1
			for pos > 0 && sim > results[pos-1].similarity {
				pos--
			}
			// Shift elements and insert
			copy(results[pos+1:], results[pos:len(results)-1]) // Shift elements to the right
			results[pos] = result{id, sim}                     // Insert new element
		}
	}

	// Extract IDs and similarities
	ids := make([]uint, len(results))
	similarities := make([]float64, len(results))

	for i, r := range results {
		ids[i] = r.id
		similarities[i] = r.similarity
	}

	return ids, similarities
}
