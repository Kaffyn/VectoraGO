package db

import (
	"context"
	"fmt"
)

// MemoryService manages the indexing of volatile and permanent knowledge using UsearchStore.
type MemoryService struct {
	store *UsearchStore
}

func NewMemoryService(ctx context.Context, dbPath string) (*MemoryService, error) {
	store, err := NewVectorStoreAtPath(dbPath)
	if err != nil {
		return nil, fmt.Errorf("memory_db_init_failed: %w", err)
	}

	return &MemoryService{
		store: store,
	}, nil
}

// StoreInsight guarda um fato no banco vetorial.
func (s *MemoryService) StoreInsight(ctx context.Context, id, content string, metadata map[string]string, embedding []float32) error {
	chunk := Chunk{
		ID:       id,
		Content:  content,
		Metadata: metadata,
		Vector:   embedding,
	}
	return s.store.UpsertChunk(ctx, "knowledge_base", chunk)
}

// SearchInsight performs local RAG over accumulated knowledge.
func (s *MemoryService) SearchInsight(ctx context.Context, query string, queryVec []float32, topK int) ([]string, error) {
	if len(queryVec) == 0 {
		return nil, fmt.Errorf("queryVec is required")
	}

	results, err := s.store.Query(ctx, "knowledge_base", queryVec, topK)
	if err != nil {
		return nil, err
	}

	var insights []string
	for _, res := range results {
		insights = append(insights, res.Content)
	}
	return insights, nil
}

// Close resources
func (s *MemoryService) Close() error {
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}
