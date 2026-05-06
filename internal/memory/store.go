package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type MemoryStore interface {
	Store(ctx context.Context, key string, value map[string]any) error
	Search(ctx context.Context, query string, topK int) ([]SearchResult, error)
	Delete(ctx context.Context, key string) error
	Size() int
}
type SearchResult struct {
	Key      string         `json:"key"`
	Value    map[string]any `json:"value"`
	Score    float64        `json:"score"`
	StoredAt time.Time      `json:"stored_at"`
}

type document struct {
	key       string
	value     map[string]any
	embedding []float64
	storedAt  time.Time
}

type InMemoryStore struct {
	mu       sync.RWMutex
	docs     map[string]*document
	order    []string
	embedder Embedder
}

// NewInMemoryStore initializes a new InMemoryStore with the given embedder.
func NewInMemoryStore(e Embedder) *InMemoryStore {
	return &InMemoryStore{
		docs:     make(map[string]*document),
		embedder: e,
	}
}

// Store embeds the key/value pair and persists it in the in-memory store.
func (s *InMemoryStore) Store(ctx context.Context, key string, value map[string]any) error {
	if key == "" {
		return fmt.Errorf("memory: key must not be empty")
	}

	if value == nil {
		return fmt.Errorf("memory: value must not be nil")
	}

	text := buildEmbedText(key, value)
	vec, err := s.embedder.Embed(ctx, text)
	if err != nil {
		return fmt.Errorf("memory: embed failed for key %q : %w", key, err)
	}

	cp := copyMap(value)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.docs[key]; !exists {
		s.order = append(s.order, key)
	}

	s.docs[key] = &document{
		key:       key,
		value:     cp,
		embedding: vec,
		storedAt:  time.Now().UTC(),
	}

	return nil
}

// Search finds the top-K most similar documents to the query string based on cosine similarity.
func (s *InMemoryStore) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("memory: query must not be empty")
	}
	if topK <= 0 {
		topK = 5
	}

	qVec, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("memory: embed query failed: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	type scored struct {
		doc   *document
		score float64
	}

	candidates := make([]scored, 0, len(s.docs))
	for _, doc := range s.docs {
		sim := cosineSimilarity(qVec, doc.embedding)
		candidates = append(candidates, scored{doc: doc, score: sim})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	if topK > len(candidates) {
		topK = len(candidates)
	}

	results := make([]SearchResult, topK)
	for i := 0; i < topK; i++ {
		results[i] = SearchResult{
			Key:      candidates[i].doc.key,
			Value:    copyMap(candidates[i].doc.value),
			Score:    candidates[i].score,
			StoredAt: candidates[i].doc.storedAt,
		}
	}
	return results, nil
}

// Delete removes a document from the store by its key.
func (s *InMemoryStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.docs, key)
	for i, k := range s.order {
		if k == key {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	return nil
}

// Size returns the total number of documents currently stored.
func (s *InMemoryStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.docs)
}

// buildEmbedText constructs a searchable string representation of a key/value pair.
func buildEmbedText(key string, value map[string]any) string {
	parts := []string{key}
	for k, v := range value {
		parts = append(parts, fmt.Sprintf("%s %v", k, v))
	}
	sort.Strings(parts[1:])
	text := ""
	for i, p := range parts {
		if i > 0 {
			text += " "
		}
		text += p
	}
	return text
}

// copyMap creates a shallow copy of a map[string]any.
func copyMap(m map[string]any) map[string]any {
	cp := make(map[string]any, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}
