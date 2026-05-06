package memory

import (
	"context"
	"crypto/md5"
	"fmt"
	"math"
	"strings"
	"sync"
)

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
	Dimensions() int
}

const mockDimensions = 64

// NewMockEmbedder returns a new MockEmbedder with default dimensions.
func NewMockEmbedder() *MockEmbedder {
	return &MockEmbedder{
		dims: mockDimensions,
	}
}

type MockEmbedder struct {
	dims int
	mu   sync.Mutex
}

// Embed converts text into a vector representation.
func (e *MockEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("Embedder: text must not be empty")
	}

	vec := e.textToVector(text)
	return vec, nil
}

// Dimensions returns the size of the embedding vector.
func (e *MockEmbedder) Dimensions() int {
	return e.dims
}

// textToVector implements a deterministic mapping from text to a pseudo-random vector.
func (e *MockEmbedder) textToVector(text string) []float64 {
	h := md5.Sum([]byte(text))

	vec := make([]float64, e.dims)
	words := strings.Fields(strings.ToLower(text))

	for i := 0; i < e.dims; i++ {
		base := float64(h[i%len(h)]) / 128.0

		wordSig := 0.0
		for j, w := range words {
			wh := md5.Sum([]byte(w))
			idx := (i + j*7) % len(wh)
			wordSig += float64(wh[idx]) / 256.0
		}

		if len(words) > 0 {
			wordSig /= float64(len(words))
		}

		vec[i] = base - 1.0 + wordSig
	}

	return vec

}

// normalise scales a vector to unit length.
func normalise(vec []float64) {
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return
	}

	for i := range vec {
		vec[i] /= norm
	}
}

// cosineSimilarity calculates the dot product of two vectors (assumed to be normalised).
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dot float64
	for i := range a {
		dot += a[i] * b[i]
	}

	return dot
}
