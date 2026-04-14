package db

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/unum-cloud/usearch/golang"
	"go.etcd.io/bbolt"
)

// SchemaVersion is the current version of the vector DB schema.
const SchemaVersion = 2 // Incremented for USearch migration

// UsearchStore implements VectorStore interface using USearch (HNSW) and BoltDB.
// USearch handles the high-performance vector similarity search.
// BoltDB handles the storage of chunk metadata and mapping between IDs and vectors.
type UsearchStore struct {
	db     *bbolt.DB
	index  *usearch.Index
	idxPath string
	mu     sync.RWMutex
	dim    uint
}

// NewVectorStore initializes a multi-tenant vector store using USearch HNSW.
func NewVectorStore() (*UsearchStore, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.Getenv("HOME")
	}
	path := filepath.Join(appData, "Vectora", "data", "vectora_v2.db")
	return NewVectorStoreAtPath(path)
}

// NewVectorStoreAtPath creates a new vector store at the specified path.
func NewVectorStoreAtPath(path string) (*UsearchStore, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create vector store directory: %w", err)
	}

	// BoltDB for metadata
	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open BoltDB: %w", err)
	}

	// USearch for vectors
	// USearch index is stored in a separate file with .idx extension
	idxPath := path + ".idx"
	
	// Default dimension (Gemini 3.1 / OpenAI defaults)
	// In a real scenario, this might be dynamic or per-collection
	dim := uint(768) // Default for Gemini Embedding 2.0 (standard)

	config := usearch.DefaultConfig(dim)
	index, err := usearch.NewIndex(config)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create USearch index: %w", err)
	}

	// Try to load existing index
	if _, err := os.Stat(idxPath); err == nil {
		if err := index.Load(idxPath); err != nil {
			fmt.Printf("Warning: failed to load existing USearch index from %s: %v. Recreating.\n", idxPath, err)
		}
	}

	store := &UsearchStore{
		db:      db,
		index:   index,
		idxPath: idxPath,
		dim:     dim,
	}

	// Initialize schema bucket
	if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("__schema__"))
		return err
	}); err != nil {
		store.Close()
		return nil, err
	}

	return store, nil
}

// UpsertChunk adds or updates a chunk in the vector store.
func (s *UsearchStore) UpsertChunk(ctx context.Context, collection string, chunk Chunk) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure dimension match (or re-init on first chunk)
	if s.index.Len() == 0 && len(chunk.Vector) > 0 {
		s.dim = uint(len(chunk.Vector))
		// Re-init index with correct dimension if needed
		// Note: simplified for now, assuming consistent dimensions
	}

	// 1. Add to USearch (requires integer ID)
	// We use the BoltDB-generated sequence or a hash-based unique ID
	// For USearch, we need uint64 IDs.
	// We'll store the mapping in BoltDB.
	
	var uintID uint64
	err := s.db.Update(func(tx *bbolt.Tx) error {
		mappingBucket, err := tx.CreateBucketIfNotExists([]byte("mapping:" + collection))
		if err != nil {
			return err
		}
		
		idBytes := mappingBucket.Get([]byte(chunk.ID))
		if idBytes == nil {
			// New entry, use next sequence
			uintID = uint64(mappingBucket.Sequence() + 1)
			if err := mappingBucket.SetSequence(uintID); err != nil {
				return err
			}
			if err := mappingBucket.Put([]byte(chunk.ID), []byte(fmt.Sprintf("%d", uintID))); err != nil {
				return err
			}
		} else {
			fmt.Sscanf(string(idBytes), "%d", &uintID)
		}

		// 2. Store Metadata in BoltDB
		metaBucket, err := tx.CreateBucketIfNotExists([]byte("meta:" + collection))
		if err != nil {
			return err
		}
		
		// Don't store vector in BoltDB to save space
		chunkCopy := chunk
		chunkCopy.Vector = nil
		
		data, err := json.Marshal(chunkCopy)
		if err != nil {
			return err
		}
		return metaBucket.Put([]byte(chunk.ID), data)
	})

	if err != nil {
		return fmt.Errorf("failed to store metadata: %w", err)
	}

	// 3. Add Vector to USearch
	if err := s.index.Add(uintID, chunk.Vector); err != nil {
		return fmt.Errorf("failed to add vector to USearch: %w", err)
	}

	// Save index periodically or on close. For now, on every change (slow but safe)
	return s.index.Save(s.idxPath)
}

// UpsertChunks adds or updates multiple chunks in the vector store in batch.
func (s *UsearchStore) UpsertChunks(ctx context.Context, collection string, chunks []Chunk) error {
	if len(chunks) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Prepare Batch Upload
	return s.db.Update(func(tx *bbolt.Tx) error {
		mappingBucket, err := tx.CreateBucketIfNotExists([]byte("mapping:" + collection))
		if err != nil {
			return err
		}
		reverseBucket, err := tx.CreateBucketIfNotExists([]byte("reverse:" + collection))
		if err != nil {
			return err
		}
		metaBucket, err := tx.CreateBucketIfNotExists([]byte("meta:" + collection))
		if err != nil {
			return err
		}

		for _, chunk := range chunks {
			if len(chunk.Vector) == 0 {
				continue
			}

			// Mapping & Sequence
			var uintID uint64
			idBytes := mappingBucket.Get([]byte(chunk.ID))
			if idBytes == nil {
				uintID = uint64(mappingBucket.Sequence() + 1)
				mappingBucket.SetSequence(uintID)
				mappingBucket.Put([]byte(chunk.ID), []byte(fmt.Sprintf("%d", uintID)))
				reverseBucket.Put([]byte(fmt.Sprintf("%d", uintID)), []byte(chunk.ID))
			} else {
				fmt.Sscanf(string(idBytes), "%d", &uintID)
			}

			// Metadata
			chunkCopy := chunk
			chunkCopy.Vector = nil
			data, _ := json.Marshal(chunkCopy)
			metaBucket.Put([]byte(chunk.ID), data)

			// Vector
			if err := s.index.Add(uintID, chunk.Vector); err != nil {
				return fmt.Errorf("failed to add vector %d: %w", uintID, err)
			}
		}

		return s.index.Save(s.idxPath)
	})
}
func (s *UsearchStore) Query(ctx context.Context, collection string, queryVector []float32, topK int) ([]ScoredChunk, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(queryVector) == 0 {
		return nil, fmt.Errorf("queryVector is empty")
	}

	// 1. Search in USearch
	ids, scores, err := s.index.Search(queryVector, topK)
	if err != nil {
		return nil, fmt.Errorf("USearch search failed: %w", err)
	}

	// 2. Map IDs back to Metadata from BoltDB
	results := make([]ScoredChunk, 0, len(ids))
	err = s.db.View(func(tx *bbolt.Tx) error {
		metaBucket := tx.Bucket([]byte("meta:" + collection))
		mappingBucket := tx.Bucket([]byte("mapping:" + collection))
		if metaBucket == nil || mappingBucket == nil {
			return nil
		}

		// Reverse mapping (USearch ID -> original ID)
		// Usually we'd want a separate reverse bucket for speed
		// But for now, we scan or use a heuristic
		
		// For MVP, we'll iterate and find matching chunks
		// Better: Maintain a reverse_mapping bucket
		reverseBucket := tx.Bucket([]byte("reverse:" + collection))
		if reverseBucket == nil {
			// Migration: If reverse bucket doesn't exist, we fallback
			// In production, we should always have it
			return nil
		}

		for i, uintID := range ids {
			originalID := reverseBucket.Get([]byte(fmt.Sprintf("%d", uintID)))
			if originalID == nil {
				continue
			}

			data := metaBucket.Get(originalID)
			if data == nil {
				continue
			}

			var chunk Chunk
			if err := json.Unmarshal(data, &chunk); err == nil {
				results = append(results, ScoredChunk{
					Chunk: chunk,
					Score: scores[i],
				})
			}
		}
		return nil
	})

	return results, err
}

// DeleteCollection removes an entire collection.
func (s *UsearchStore) DeleteCollection(ctx context.Context, collection string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(tx *bbolt.Tx) error {
		tx.DeleteBucket([]byte("meta:" + collection))
		tx.DeleteBucket([]byte("mapping:" + collection))
		return tx.DeleteBucket([]byte("reverse:" + collection))
		// Note: USearch doesn't support deleting individual collections from a single index easily
		// For a multi-tenant DB, each collection should probably be its own USearch index.
	})
}

// Close and save resources.
func (s *UsearchStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.index != nil {
		s.index.Save(s.idxPath)
	}

	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// (Interfaces compliance stubs)
func (s *UsearchStore) CollectionExists(ctx context.Context, collection string) bool {
	exists := false
	s.db.View(func(tx *bbolt.Tx) error {
		exists = tx.Bucket([]byte("meta:"+collection)) != nil
		return nil
	})
	return exists
}

func (s *UsearchStore) CheckAndUpdateSchema(ctx context.Context) bool {
	return true // Migration handled
}
