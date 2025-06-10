package embeddings

import "github.com/matst80/slask-finder/pkg/types"

// ConvertToFloat64Embeddings converts the Ollama embeddings ([]float32)
// to the float64 format used by the EmbeddingsIndex
func ConvertToFloat64Embeddings(emb types.Embeddings) []float64 {
	result := make([]float64, len(emb))
	for i, v := range emb {
		result[i] = float64(v)
	}
	return result
}

// ConvertToTypesEmbeddings converts a []float64 slice to types.Embeddings ([]float32)
func ConvertToTypesEmbeddings(emb []float64) types.Embeddings {
	result := make(types.Embeddings, len(emb))
	for i, v := range emb {
		result[i] = float32(v)
	}
	return result
}
