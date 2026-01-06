package store

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/austiecodes/gomor/internal/consts"
	"github.com/austiecodes/gomor/internal/memory/memtypes"
	"github.com/austiecodes/gomor/internal/memory/memutils"
)

//go:embed sql/schema.sql
var schemaSQL string

//go:embed sql/queries.sql
var queriesSQL string

// queries holds parsed SQL queries by name
var queries map[string]string

func init() {
	queries = parseQueries(queriesSQL)
}

// parseQueries extracts named queries from a SQL file.
// Queries are marked with "-- name: QueryName" comments.
func parseQueries(content string) map[string]string {
	result := make(map[string]string)
	re := regexp.MustCompile(`(?m)^--\s*name:\s*(\w+)\s*$`)
	matches := re.FindAllStringSubmatchIndex(content, -1)

	for i, match := range matches {
		name := content[match[2]:match[3]]
		start := match[1]
		end := len(content)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		query := strings.TrimSpace(content[start:end])
		result[name] = query
	}

	return result
}

// Re-export types from memtypes for convenience
type MemoryItem = memtypes.MemoryItem
type MemorySource = memtypes.MemorySource
type HistoryItem = memtypes.HistoryItem
type SearchResult = memtypes.SearchResult
type MemoryFTSResult = memtypes.MemoryFTSResult
type HistorySearchResult = memtypes.HistorySearchResult

const (
	SourceExplicit  = memtypes.SourceExplicit
	SourceExtracted = memtypes.SourceExtracted
)

// Re-export vector utils from memutils for convenience
var (
	NormalizeVector = memutils.NormalizeVector
	DotProduct      = memutils.DotProduct
	VectorToBytes   = memutils.VectorToBytes
	BytesToVector   = memutils.BytesToVector
)

// Store manages memory and history persistence in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a new memory store, initializing the database if needed.
func NewStore() (*Store, error) {
	dbPath, err := getDBPath()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open memory database: %w", err)
	}

	store := &Store{db: db}
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// NewStoreWithDB creates a new memory store with a provided database connection.
// This is primarily used for testing.
func NewStoreWithDB(db *sql.DB) (*Store, error) {
	store := &Store{db: db}
	if err := store.initSchema(); err != nil {
		return nil, err
	}
	return store, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// getDBPath returns the path to the memory database file.
func getDBPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	goaDir := filepath.Join(homeDir, consts.GoaDir)
	if err := os.MkdirAll(goaDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create gomor directory: %w", err)
	}

	return filepath.Join(goaDir, "memory.db"), nil
}

// initSchema creates the database tables if they don't exist.
func (s *Store) initSchema() error {
	if _, err := s.db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}
	return nil
}

// SaveMemory saves a new memory item with its embedding.
func (s *Store) SaveMemory(item *MemoryItem) error {
	if item.ID == "" {
		item.ID = uuid.New().String()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}

	tagsJSON, err := json.Marshal(item.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	embeddingBytes := VectorToBytes(item.Embedding)

	_, err = s.db.Exec(queries["InsertMemory"],
		item.ID, item.Text, string(tagsJSON), string(item.Source), item.Confidence,
		item.CreatedAt.Unix(), item.Provider, item.ModelID, item.Dim, embeddingBytes)

	if err != nil {
		return fmt.Errorf("failed to save memory: %w", err)
	}

	return nil
}

// UpdateMemoryEmbedding updates the embedding for a specific memory.
func (s *Store) UpdateMemoryEmbedding(id string, embedding []float32, modelID string, dim int, provider string) error {
	embeddingBytes := VectorToBytes(embedding)
	_, err := s.db.Exec(queries["UpdateMemoryEmbedding"], embeddingBytes, modelID, dim, provider, id)
	if err != nil {
		return fmt.Errorf("failed to update memory embedding: %w", err)
	}
	return nil
}

// GetAllMemories returns all memory items (for vector search).
func (s *Store) GetAllMemories() ([]MemoryItem, error) {
	rows, err := s.db.Query(queries["SelectAllMemories"])
	if err != nil {
		return nil, fmt.Errorf("failed to query memories: %w", err)
	}
	defer rows.Close()

	var memories []MemoryItem
	for rows.Next() {
		var item MemoryItem
		var tagsJSON string
		var createdAtUnix int64
		var embeddingBytes []byte
		var source string

		err := rows.Scan(&item.ID, &item.Text, &tagsJSON, &source, &item.Confidence,
			&createdAtUnix, &item.Provider, &item.ModelID, &item.Dim, &embeddingBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memory row: %w", err)
		}

		item.Source = MemorySource(source)
		item.CreatedAt = time.Unix(createdAtUnix, 0)
		item.Embedding = BytesToVector(embeddingBytes)

		if err := json.Unmarshal([]byte(tagsJSON), &item.Tags); err != nil {
			item.Tags = nil // ignore malformed tags
		}

		memories = append(memories, item)
	}

	return memories, rows.Err()
}

// SearchMemories performs vector similarity search on memories.
// Returns top K results with similarity >= minSimilarity.
func (s *Store) SearchMemories(queryEmbedding []float32, topK int, minSimilarity float64) ([]SearchResult, error) {
	memories, err := s.GetAllMemories()
	if err != nil {
		return nil, err
	}

	// Normalize query embedding for cosine similarity via dot product
	normalizedQuery := NormalizeVector(queryEmbedding)

	// Calculate similarities
	var results []SearchResult
	for _, mem := range memories {
		// Embeddings are stored normalized, so dot product = cosine similarity
		similarity := DotProduct(normalizedQuery, mem.Embedding)
		if similarity >= minSimilarity {
			results = append(results, SearchResult{
				Item:       mem,
				Similarity: similarity,
			})
		}
	}

	// Sort by similarity descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	// Return top K
	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// DeleteMemory deletes a memory by ID.
func (s *Store) DeleteMemory(id string) error {
	_, err := s.db.Exec(queries["DeleteMemory"], id)
	return err
}

// SearchMemoriesFTS performs full-text search on memory text.
// Returns top K results ordered by FTS rank.
func (s *Store) SearchMemoriesFTS(query string, topK int) ([]MemoryFTSResult, error) {
	rows, err := s.db.Query(queries["SearchMemoriesFTS"], query, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to search memories FTS: %w", err)
	}
	defer rows.Close()

	var results []MemoryFTSResult
	for rows.Next() {
		var item MemoryItem
		var result MemoryFTSResult
		var tagsJSON string
		var createdAtUnix int64
		var embeddingBytes []byte
		var source string

		err := rows.Scan(&item.ID, &item.Text, &tagsJSON, &source, &item.Confidence,
			&createdAtUnix, &item.Provider, &item.ModelID, &item.Dim, &embeddingBytes,
			&result.Snippet, &result.Rank)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memory FTS row: %w", err)
		}

		item.Source = MemorySource(source)
		item.CreatedAt = time.Unix(createdAtUnix, 0)
		item.Embedding = BytesToVector(embeddingBytes)

		if err := json.Unmarshal([]byte(tagsJSON), &item.Tags); err != nil {
			item.Tags = nil // ignore malformed tags
		}

		result.Item = item
		results = append(results, result)
	}

	return results, rows.Err()
}

// SaveHistory saves a new history item.
func (s *Store) SaveHistory(item *HistoryItem) error {
	if item.ID == "" {
		item.ID = uuid.New().String()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}

	_, err := s.db.Exec(queries["InsertHistory"],
		item.ID, item.Role, item.Content, item.CreatedAt.Unix(), item.SessionID)

	if err != nil {
		return fmt.Errorf("failed to save history: %w", err)
	}

	return nil
}

// SearchHistory performs full-text search on history content.
// Returns top K results ordered by FTS rank.
func (s *Store) SearchHistory(query string, topK int) ([]HistorySearchResult, error) {
	rows, err := s.db.Query(queries["SearchHistoryFTS"], query, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to search history: %w", err)
	}
	defer rows.Close()

	var results []HistorySearchResult
	for rows.Next() {
		var item HistoryItem
		var result HistorySearchResult
		var createdAtUnix int64
		var sessionID sql.NullString

		err := rows.Scan(&item.ID, &item.Role, &item.Content, &createdAtUnix,
			&sessionID, &result.Snippet, &result.Rank)
		if err != nil {
			return nil, fmt.Errorf("failed to scan history row: %w", err)
		}

		item.CreatedAt = time.Unix(createdAtUnix, 0)
		if sessionID.Valid {
			item.SessionID = sessionID.String
		}

		result.Item = item
		results = append(results, result)
	}

	return results, rows.Err()
}

// GetRecentHistory returns the most recent history items.
func (s *Store) GetRecentHistory(limit int) ([]HistoryItem, error) {
	rows, err := s.db.Query(queries["SelectRecentHistory"], limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent history: %w", err)
	}
	defer rows.Close()

	var items []HistoryItem
	for rows.Next() {
		var item HistoryItem
		var createdAtUnix int64
		var sessionID sql.NullString

		err := rows.Scan(&item.ID, &item.Role, &item.Content, &createdAtUnix, &sessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan history row: %w", err)
		}

		item.CreatedAt = time.Unix(createdAtUnix, 0)
		if sessionID.Valid {
			item.SessionID = sessionID.String
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

// ClearHistory deletes all history items.
func (s *Store) ClearHistory() error {
	_, err := s.db.Exec(queries["ClearHistory"])
	return err
}

// ClearMemories deletes all memory items.
func (s *Store) ClearMemories() error {
	_, err := s.db.Exec(queries["ClearMemories"])
	return err
}
