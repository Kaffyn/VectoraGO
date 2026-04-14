package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Kaffyn/Vectora/internal/storage/db"
	"github.com/Kaffyn/Vectora/internal/storage/infra"
	"github.com/Kaffyn/Vectora/internal/llm"
	"github.com/Kaffyn/Vectora/internal/config/policies"
)

type EmbedJobConfig struct {
	RootPath       string
	Include        string
	Exclude        string
	Workspace      string
	Force          bool
	CollectionName string
}

type EmbedProgress struct {
	CurrentIdx      int     `json:"current_idx"`
	TotalFiles      int     `json:"total_files"`
	DisplayPath     string  `json:"display_path"`
	ElapsedSeconds  float64 `json:"elapsed_seconds"`
	FilesSkipped    int     `json:"files_skipped"`
	FilesAlready    int     `json:"files_already"`
	TotalChunks     int     `json:"total_chunks"`
	TotalEmbedded   int     `json:"total_embedded"`
	TotalErrors     int     `json:"total_errors"`
	IsComplete      bool    `json:"is_complete"`
	CurrentFilePath string  `json:"current_file_path"`
	HasError        bool    `json:"has_error"`
	ErrorMsg        string  `json:"error_msg"`
	FileChunks      int     `json:"file_chunks"`
}

var (
	activeJobs     = make(map[string]bool)
	activeJobsLock sync.Mutex
)

// RunEmbedJob contains the entire directory walking and chunking logic.
func RunEmbedJob(ctx context.Context, cfg EmbedJobConfig, kvStore db.KVStore, storage db.VectorStore, provider llm.Provider, onProgress func(EmbedProgress)) {
	activeJobsLock.Lock()
	if activeJobs[cfg.Workspace] {
		activeJobsLock.Unlock()
		onProgress(EmbedProgress{IsComplete: true, HasError: true, ErrorMsg: "A job is already running for this workspace"})
		return
	}
	activeJobs[cfg.Workspace] = true
	activeJobsLock.Unlock()

	defer func() {
		activeJobsLock.Lock()
		delete(activeJobs, cfg.Workspace)
		activeJobsLock.Unlock()
	}()

	absPath, err := filepath.Abs(cfg.RootPath)
	if err != nil {
		onProgress(EmbedProgress{IsComplete: true, HasError: true, ErrorMsg: fmt.Sprintf("invalid path: %v", err)})
		return
	}

	// Register workspace path mapping for `workspace ls` display
	registerWorkspacePath(cfg.CollectionName, absPath)

	guardian := policies.NewGuardian(absPath)
	if provider == nil || !provider.IsConfigured() {
		onProgress(EmbedProgress{IsComplete: true, HasError: true, ErrorMsg: "No LLM provider configured. Set it in .env or via API."})
		return
	}

	// Parse includes and excludes similarly
	var includePatterns []string
	if cfg.Include != "" {
		includePatterns = strings.Split(cfg.Include, ",")
		for i, p := range includePatterns {
			includePatterns[i] = strings.TrimSpace(p)
		}
	}

	var excludePatterns []string
	var unignorePatterns []string
	if cfg.Exclude != "" {
		parts := strings.Split(cfg.Exclude, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "!") {
				unignorePatterns = append(unignorePatterns, strings.TrimPrefix(p, "!"))
			} else {
				excludePatterns = append(excludePatterns, p)
			}
		}
	}

	ignorePath := filepath.Join(absPath, ".embedignore")
	if data, err := os.ReadFile(ignorePath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "!") {
				unignorePatterns = append(unignorePatterns, strings.TrimPrefix(line, "!"))
			} else {
				excludePatterns = append(excludePatterns, line)
			}
		}
	}

	var filesToEmbed []string
	var filesSkipped int
	var filesAlreadyIndexed int

	err = filepath.WalkDir(absPath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			relPath, _ := filepath.Rel(absPath, path)
			relPath = filepath.ToSlash(relPath)
			name := d.Name()

			if guardian.IsExcludedDir(name) {
				return filepath.SkipDir
			}

			isIgnored := false
			for _, pattern := range excludePatterns {
				if m, _ := filepath.Match(pattern, relPath); m {
					isIgnored = true
				} else if m, _ := filepath.Match(pattern, name); m {
					isIgnored = true
				} else if strings.Contains(pattern, "/") {
					if strings.HasPrefix(relPath, pattern) {
						isIgnored = true
					} else {
						m, _ := filepath.Match(pattern+"/*", relPath)
						if m {
							isIgnored = true
						}
					}
				}
			}

			if isIgnored {
				for _, pattern := range unignorePatterns {
					if m, _ := filepath.Match(pattern, relPath); m {
						isIgnored = false
					} else if m, _ := filepath.Match(pattern, name); m {
						isIgnored = false
					} else if strings.Contains(pattern, "/") {
						if strings.HasPrefix(relPath, pattern) {
							isIgnored = false
						} else {
							m, _ := filepath.Match(pattern+"/*", relPath)
							if m {
								isIgnored = false
							}
						}
					}
				}
			}

			if isIgnored {
				return filepath.SkipDir
			}
			return nil
		}

		if guardian.IsProtected(path) {
			filesSkipped++
			return nil
		}

		if len(includePatterns) > 0 {
			name := d.Name()
			matched := false
			for _, pattern := range includePatterns {
				if m, _ := filepath.Match(pattern, name); m {
					matched = true
					break
				}
			}
			if !matched {
				filesSkipped++
				return nil
			}
		}

		relPath, _ := filepath.Rel(absPath, path)
		relPath = filepath.ToSlash(relPath)
		name := d.Name()

		isIgnored := false
		for _, pattern := range excludePatterns {
			if m, _ := filepath.Match(pattern, relPath); m {
				isIgnored = true
			} else if m, _ := filepath.Match(pattern, name); m {
				isIgnored = true
			} else if strings.Contains(pattern, "/") {
				if strings.HasPrefix(relPath, pattern) {
					isIgnored = true
				} else {
					m, _ := filepath.Match(pattern+"/*", relPath)
					if m {
						isIgnored = true
					}
				}
			}
		}

		if isIgnored {
			for _, pattern := range unignorePatterns {
				if m, _ := filepath.Match(pattern, relPath); m {
					isIgnored = false
				} else if m, _ := filepath.Match(pattern, name); m {
					isIgnored = false
				} else if strings.Contains(pattern, "/") {
					if strings.HasPrefix(relPath, pattern) {
						isIgnored = false
					} else {
						m, _ := filepath.Match(pattern+"/*", relPath)
						if m {
							isIgnored = false
						}
					}
				}
			}
		}

		if isIgnored {
			filesSkipped++
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			filesSkipped++
			return nil
		}

		if !cfg.Force {
			existing, _ := kvStore.Get(ctx, "file_index", relPath)
			if existing != nil {
				var entry db.FileIndexEntry
				if err := json.Unmarshal(existing, &entry); err == nil {
					contentHash := db.CalculateHash(string(content))
					if entry.ContentHash == contentHash {
						filesAlreadyIndexed++
						return nil
					}
				}
			}
		}

		filesToEmbed = append(filesToEmbed, path)
		return nil
	})

	if err != nil {
		onProgress(EmbedProgress{IsComplete: true, HasError: true, ErrorMsg: fmt.Sprintf("walk error: %v", err)})
		return
	}

	if len(filesToEmbed) == 0 {
		onProgress(EmbedProgress{
			IsComplete:   true,
			FilesSkipped: filesSkipped,
			FilesAlready: filesAlreadyIndexed,
			TotalFiles:   0,
		})
		return
	}

	infra.NotifyOS("Vectora Indexing", fmt.Sprintf("Started indexing %d files", len(filesToEmbed)))

	totalEmbedded := 0
	totalChunks := 0
	totalErrors := 0
	startTime := time.Now()

	// Worker pool for parallel embedding
	const numWorkers = 4
	workQueue := make(chan embeddingWorkItem, 16) // buffered channel
	resultQueue := make(chan embeddingResult, 16)
	var wg sync.WaitGroup

	// Start worker pool
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go embedWorker(ctx, w, workQueue, resultQueue, &wg, cfg.CollectionName, absPath, storage, provider)
	}

	// Process results concurrently
	var resultWg sync.WaitGroup
	resultWg.Add(1)
	go func() {
		defer resultWg.Done()
		processEmbeddingResults(ctx, resultQueue, kvStore, &totalEmbedded, &totalChunks, &totalErrors, startTime, onProgress, len(filesToEmbed))
	}()

	// Fan-out: send files to worker pool
	for i, filePath := range filesToEmbed {
		select {
		case <-ctx.Done():
			close(workQueue)
			onProgress(EmbedProgress{IsComplete: true, HasError: true, ErrorMsg: "context canceled"})
			return
		default:
		}

		relPath, _ := filepath.Rel(absPath, filePath)
		displayPath := relPath
		if len(displayPath) > 40 {
			displayPath = "..." + displayPath[len(displayPath)-37:]
		}

		onProgress(EmbedProgress{
			CurrentIdx:      i,
			TotalFiles:      len(filesToEmbed),
			DisplayPath:     displayPath,
			ElapsedSeconds:  time.Since(startTime).Seconds(),
			CurrentFilePath: relPath,
		})

		content, err := os.ReadFile(filePath)
		if err != nil {
			totalErrors++
			onProgress(EmbedProgress{
				CurrentIdx: i, TotalFiles: len(filesToEmbed), DisplayPath: displayPath,
				ElapsedSeconds: time.Since(startTime).Seconds(), CurrentFilePath: relPath,
				HasError: true, ErrorMsg: "read error", FileChunks: 0,
				TotalEmbedded: totalEmbedded, TotalChunks: totalChunks, TotalErrors: totalErrors,
			})
			continue
		}

		chunks := chunkContent(string(content), 800, 100)
		language := detectLanguage(filePath)

		// Send work item to queue
		workQueue <- embeddingWorkItem{
			filePath:    filePath,
			relPath:     relPath,
			displayPath: displayPath,
			content:     string(content),
			chunks:      chunks,
			language:    language,
			fileIdx:     i,
			totalFiles:  len(filesToEmbed),
		}
	}

	// Signal workers that no more work
	close(workQueue)

	// Wait for all workers to finish
	wg.Wait()

	// Close result queue and wait for result processor
	close(resultQueue)
	resultWg.Wait()

	infra.NotifyOS("Vectora Indexing", fmt.Sprintf("Completed indexing. %d files embedded.", totalEmbedded))

	onProgress(EmbedProgress{
		IsComplete:     true,
		FilesSkipped:   filesSkipped,
		FilesAlready:   filesAlreadyIndexed,
		TotalFiles:     len(filesToEmbed),
		TotalEmbedded:  totalEmbedded,
		TotalChunks:    totalChunks,
		TotalErrors:    totalErrors,
		ElapsedSeconds: time.Since(startTime).Seconds(),
	})
}

// Worker pool types and functions

type embeddingWorkItem struct {
	filePath    string
	relPath     string
	displayPath string
	content     string
	chunks      []string
	language    string
	fileIdx     int
	totalFiles  int
}

type embeddingResult struct {
	filePath    string
	relPath     string
	displayPath string
	fileIdx     int
	totalFiles  int
	content     string
	fileChunks  int
	hasError    bool
	errorMsg    string
	startTime   time.Time
}

// embedWorker processes files from work queue and sends results to result queue
func embedWorker(ctx context.Context, workerID int, workQueue chan embeddingWorkItem, resultQueue chan embeddingResult, wg *sync.WaitGroup, collectionName string, absPath string, storage db.VectorStore, provider llm.Provider) {
	defer wg.Done()

	for item := range workQueue {
		select {
		case <-ctx.Done():
			return
		default:
		}

		fileChunks := 0
		hasError := false
		var errorMsg string

		// Process chunks sequentially per file, but files are parallel across workers
		for j, chunk := range item.chunks {
			vec, err := provider.Embed(ctx, chunk, "")
			if err != nil {
				hasError = true
				errorMsg = fmt.Sprintf("embed error: %v", err)
				break
			}

			docID := fmt.Sprintf("%s:%d", item.relPath, j)
			err = storage.UpsertChunk(ctx, collectionName, db.Chunk{
				ID:       docID,
				Content:  chunk,
				Metadata: map[string]string{"source": item.relPath, "filename": filepath.Base(item.filePath), "language": item.language},
				Vector:   vec,
			})
			if err != nil {
				hasError = true
				errorMsg = fmt.Sprintf("store error: %v", err)
				break
			}
			fileChunks++
		}

		resultQueue <- embeddingResult{
			filePath:    item.filePath,
			relPath:     item.relPath,
			displayPath: item.displayPath,
			fileIdx:     item.fileIdx,
			totalFiles:  item.totalFiles,
			content:     item.content,
			fileChunks:  fileChunks,
			hasError:    hasError,
			errorMsg:    errorMsg,
			startTime:   time.Now(),
		}
	}
}

// processEmbeddingResults aggregates results from workers and updates progress
func processEmbeddingResults(ctx context.Context, resultQueue chan embeddingResult, kvStore db.KVStore, totalEmbedded *int, totalChunks *int, totalErrors *int, startTime time.Time, onProgress func(EmbedProgress), totalFiles int) {
	for result := range resultQueue {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if result.hasError {
			*totalErrors++
			onProgress(EmbedProgress{
				CurrentIdx: result.fileIdx, TotalFiles: totalFiles, DisplayPath: result.displayPath,
				ElapsedSeconds: time.Since(startTime).Seconds(), CurrentFilePath: result.relPath,
				HasError: true, ErrorMsg: result.errorMsg, FileChunks: result.fileChunks,
				TotalEmbedded: *totalEmbedded, TotalChunks: *totalChunks, TotalErrors: *totalErrors,
			})
		} else {
			if result.fileChunks > 0 {
				contentHash := db.CalculateHash(result.content)
				entry := db.FileIndexEntry{
					AbsolutePath: result.filePath,
					ContentHash:  contentHash,
					SizeBytes:    int64(len(result.content)),
				}
				entryBytes, _ := json.Marshal(entry)
				kvStore.Set(ctx, "file_index", result.relPath, entryBytes)

				*totalEmbedded++
				*totalChunks += result.fileChunks
			}

			onProgress(EmbedProgress{
				CurrentIdx: result.fileIdx, TotalFiles: totalFiles, DisplayPath: result.displayPath,
				ElapsedSeconds: time.Since(startTime).Seconds(), CurrentFilePath: result.relPath,
				HasError: false, FileChunks: result.fileChunks,
				TotalEmbedded: *totalEmbedded, TotalChunks: *totalChunks, TotalErrors: *totalErrors,
			})
		}
	}
}

// detectLanguage determines file language from extension
func detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".ts":
		return "javascript"
	case ".md":
		return "markdown"
	default:
		return "text"
	}
}

// Retain local chunk definition since chunking is local per file.
func chunkContent(content string, maxTokens int, overlap int) []string {
	if len(content) == 0 {
		return []string{}
	}
	maxChars := maxTokens * 4
	overlapChars := overlap * 4
	runes := []rune(content)
	totalRunes := len(runes)

	if maxChars >= totalRunes {
		return []string{content}
	}

	var chunks []string
	start := 0
	for start < totalRunes {
		end := start + maxChars
		if end > totalRunes {
			end = totalRunes
		}
		chunkEnd := end
		if end < totalRunes {
			for i := end; i < totalRunes && i < end+maxChars/2; i++ {
				if runes[i] == '\n' {
					chunkEnd = i + 1
					break
				}
			}
		}
		chunks = append(chunks, string(runes[start:chunkEnd]))
		start = chunkEnd - overlapChars
		if start < 0 {
			start = 0
		}
		if start >= chunkEnd {
			break
		}
	}
	return chunks
}
