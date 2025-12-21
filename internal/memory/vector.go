package memory

import (
	"encoding/binary"
	"math"
)

// NormalizeVector normalizes a vector to unit length (L2 norm = 1).
// This allows using dot product for cosine similarity.
func NormalizeVector(v []float32) []float32 {
	if len(v) == 0 {
		return v
	}

	var sumSquares float64
	for _, val := range v {
		sumSquares += float64(val) * float64(val)
	}

	norm := math.Sqrt(sumSquares)
	if norm == 0 {
		return v
	}

	normalized := make([]float32, len(v))
	for i, val := range v {
		normalized[i] = float32(float64(val) / norm)
	}

	return normalized
}

// DotProduct computes the dot product of two vectors.
// For normalized vectors, this equals cosine similarity.
func DotProduct(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var sum float64
	for i := range a {
		sum += float64(a[i]) * float64(b[i])
	}

	return sum
}

// CosineSimilarity computes the cosine similarity between two vectors.
// Returns a value in [-1, 1] where 1 means identical direction.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// VectorToBytes converts a float32 slice to bytes (little-endian).
func VectorToBytes(v []float32) []byte {
	buf := make([]byte, len(v)*4)
	for i, val := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(val))
	}
	return buf
}

// BytesToVector converts bytes (little-endian) to a float32 slice.
func BytesToVector(b []byte) []float32 {
	if len(b)%4 != 0 {
		return nil
	}

	v := make([]float32, len(b)/4)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return v
}

