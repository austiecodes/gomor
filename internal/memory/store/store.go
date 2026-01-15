package store

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/austiecodes/gomor/internal/memory/decay"
	"github.com/austiecodes/gomor/internal/memory/memtypes"
	"github.com/austiecodes/gomor/internal/memory/memutils"
	"github.com/austiecodes/gomor/internal/utils"
)

// Re-export types from memtypes for convenience
type MemoryItem = memtypes.MemoryItem
type MemorySource = memtypes.MemorySource
type HistoryItem = memtypes.HistoryItem
type SearchResult = memtypes.SearchResult
type MemoryFTSResult = memtypes.MemoryFTSResult
type HistorySearchResult = memtypes.HistorySearchResult

// Re-export constants from memtypes for convenience
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
	dbPath, err := utils.GetDBPath()
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

// initSchema creates the database tables if they don't exist.
func (s *Store) initSchema() error {
	if _, err := s.db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}
	if err := s.ensureMemoryColumns(); err != nil {
		return err
	}
	if err := s.rebuildFTSIndexes(); err != nil {
		return err
	}
	if err := s.backfillMemoryDecayFields(); err != nil {
		return err
	}
	return nil
}

func (s *Store) ensureMemoryColumns() error {
	columns, err := s.memoryColumns()
	if err != nil {
		return fmt.Errorf("failed to inspect memory schema: %w", err)
	}

	if !columns["confidence"] {
		if _, err := s.db.Exec(`ALTER TABLE memories ADD COLUMN confidence REAL NOT NULL DEFAULT 0;`); err != nil {
			return fmt.Errorf("failed to add memories.confidence column: %w", err)
		}
	}
	if !columns["stability_days"] {
		if _, err := s.db.Exec(`ALTER TABLE memories ADD COLUMN stability_days REAL NOT NULL DEFAULT 0;`); err != nil {
			return fmt.Errorf("failed to add memories.stability_days column: %w", err)
		}
	}
	if !columns["last_retrieved_at"] {
		if _, err := s.db.Exec(`ALTER TABLE memories ADD COLUMN last_retrieved_at INTEGER;`); err != nil {
			return fmt.Errorf("failed to add memories.last_retrieved_at column: %w", err)
		}
	}

	return nil
}

func (s *Store) memoryColumns() (map[string]bool, error) {
	rows, err := s.db.Query(`PRAGMA table_info(memories)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &primaryKey); err != nil {
			return nil, err
		}
		columns[name] = true
	}

	return columns, rows.Err()
}

func (s *Store) backfillMemoryDecayFields() error {
	if _, err := s.db.Exec(
		`UPDATE memories
		 SET confidence = CASE
		     WHEN source = ? THEN ?
		     ELSE ?
		 END
		 WHERE confidence IS NULL OR confidence <= 0`,
		string(SourceExplicit),
		decay.DefaultConfidence(SourceExplicit),
		decay.DefaultConfidence(SourceExtracted),
	); err != nil {
		return fmt.Errorf("failed to backfill memory confidence: %w", err)
	}

	if _, err := s.db.Exec(
		`UPDATE memories
		 SET stability_days = CASE
		     WHEN source = ? THEN ?
		     ELSE ?
		 END
		 WHERE stability_days IS NULL OR stability_days <= 0`,
		string(SourceExplicit),
		decay.DefaultStabilityDays(SourceExplicit),
		decay.DefaultStabilityDays(SourceExtracted),
	); err != nil {
		return fmt.Errorf("failed to backfill memory stability days: %w", err)
	}

	return nil
}

func (s *Store) rebuildFTSIndexes() error {
	if _, err := s.db.Exec(`INSERT INTO memories_fts(memories_fts) VALUES('rebuild');`); err != nil {
		return fmt.Errorf("failed to rebuild memories FTS index: %w", err)
	}
	if _, err := s.db.Exec(`INSERT INTO history_fts(history_fts) VALUES('rebuild');`); err != nil {
		return fmt.Errorf("failed to rebuild history FTS index: %w", err)
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
	if item.Confidence <= 0 {
		item.Confidence = decay.DefaultConfidence(item.Source)
	}
	if item.StabilityDays <= 0 {
		item.StabilityDays = decay.DefaultStabilityDays(item.Source)
	}

	tagsJSON, err := json.Marshal(item.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	embeddingBytes := VectorToBytes(item.Embedding)
	var lastRetrievedAt any
	if item.LastRetrievedAt != nil {
		lastRetrievedAt = item.LastRetrievedAt.Unix()
	}

	_, err = s.db.Exec(insertMemorySQL,
		item.ID, item.Text, string(tagsJSON), string(item.Source),
		item.CreatedAt.Unix(), item.Confidence, item.StabilityDays, lastRetrievedAt,
		item.Provider, item.ModelID, item.Dim, embeddingBytes)

	if err != nil {
		return fmt.Errorf("failed to save memory: %w", err)
	}

	return nil
}

// UpdateMemoryEmbedding updates the embedding for a specific memory.
func (s *Store) UpdateMemoryEmbedding(id string, embedding []float32, modelID string, dim int, provider string) error {
	embeddingBytes := VectorToBytes(embedding)
	_, err := s.db.Exec(updateMemoryEmbeddingSQL, embeddingBytes, modelID, dim, provider, id)
	if err != nil {
		return fmt.Errorf("failed to update memory embedding: %w", err)
	}
	return nil
}

// GetAllMemories returns all memory items (for vector search).
func (s *Store) GetAllMemories() ([]MemoryItem, error) {
	rows, err := s.db.Query(selectAllMemoriesSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to query memories: %w", err)
	}
	defer rows.Close()

	var memories []MemoryItem
	for rows.Next() {
		var item MemoryItem
		var tagsJSON string
		var createdAtUnix int64
		var lastRetrievedAtUnix sql.NullInt64
		var embeddingBytes []byte
		var source string

		err := rows.Scan(&item.ID, &item.Text, &tagsJSON, &source,
			&createdAtUnix, &item.Confidence, &item.StabilityDays, &lastRetrievedAtUnix,
			&item.Provider, &item.ModelID, &item.Dim, &embeddingBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memory row: %w", err)
		}

		item.Source = MemorySource(source)
		item.CreatedAt = time.Unix(createdAtUnix, 0)
		if lastRetrievedAtUnix.Valid {
			lastRetrievedAt := time.Unix(lastRetrievedAtUnix.Int64, 0)
			item.LastRetrievedAt = &lastRetrievedAt
		}
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

// UpdateMemoryDecay updates confidence, stability, and retrieval time for a memory.
func (s *Store) UpdateMemoryDecay(id string, confidence float64, stabilityDays float64, lastRetrievedAt *time.Time) error {
	var lastRetrievedAtUnix any
	if lastRetrievedAt != nil {
		lastRetrievedAtUnix = lastRetrievedAt.Unix()
	}

	_, err := s.db.Exec(updateMemoryDecaySQL, confidence, stabilityDays, lastRetrievedAtUnix, id)
	if err != nil {
		return fmt.Errorf("failed to update memory decay: %w", err)
	}
	return nil
}

// DeleteMemory deletes a memory by ID.
func (s *Store) DeleteMemory(id string) error {
	_, err := s.DeleteMemoryByID(id)
	return err
}

// DeleteMemoryByID deletes a memory by ID and reports whether a row was removed.
func (s *Store) DeleteMemoryByID(id string) (bool, error) {
	result, err := s.db.Exec(deleteMemorySQL, id)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

// SearchMemoriesFTS performs full-text search on memory text.
// Returns top K results ordered by FTS rank.
func (s *Store) SearchMemoriesFTS(query string, topK int) ([]MemoryFTSResult, error) {
	rows, err := s.db.Query(searchMemoriesFTSSQL, query, topK)
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
		var lastRetrievedAtUnix sql.NullInt64
		var embeddingBytes []byte
		var source string

		err := rows.Scan(&item.ID, &item.Text, &tagsJSON, &source,
			&createdAtUnix, &item.Confidence, &item.StabilityDays, &lastRetrievedAtUnix,
			&item.Provider, &item.ModelID, &item.Dim, &embeddingBytes,
			&result.Snippet, &result.Rank)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memory FTS row: %w", err)
		}

		item.Source = MemorySource(source)
		item.CreatedAt = time.Unix(createdAtUnix, 0)
		if lastRetrievedAtUnix.Valid {
			lastRetrievedAt := time.Unix(lastRetrievedAtUnix.Int64, 0)
			item.LastRetrievedAt = &lastRetrievedAt
		}
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

	_, err := s.db.Exec(insertHistorySQL,
		item.ID, item.Role, item.Content, item.CreatedAt.Unix(), item.SessionID)

	if err != nil {
		return fmt.Errorf("failed to save history: %w", err)
	}

	return nil
}

// SearchHistory performs full-text search on history content.
// Returns top K results ordered by FTS rank.
func (s *Store) SearchHistory(query string, topK int) ([]HistorySearchResult, error) {
	rows, err := s.db.Query(searchHistoryFTSSQL, query, topK)
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
	rows, err := s.db.Query(selectRecentHistorySQL, limit)
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
	_, err := s.db.Exec(clearHistorySQL)
	return err
}

// ClearMemories deletes all memory items.
func (s *Store) ClearMemories() error {
	_, err := s.db.Exec(clearMemoriesSQL)
	return err
}
