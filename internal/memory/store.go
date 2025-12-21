package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/austiecodes/goa/internal/consts"
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
		return "", fmt.Errorf("failed to create goa directory: %w", err)
	}

	return filepath.Join(goaDir, "memory.db"), nil
}

// initSchema creates the database tables if they don't exist.
func (s *Store) initSchema() error {
	// Create memories table for preference/fact storage with embeddings
	memoriesSchema := `
	CREATE TABLE IF NOT EXISTS memories (
		id TEXT PRIMARY KEY,
		text TEXT NOT NULL,
		tags TEXT,
		source TEXT NOT NULL,
		confidence REAL NOT NULL,
		created_at INTEGER NOT NULL,
		provider TEXT NOT NULL,
		model_id TEXT NOT NULL,
		dim INTEGER NOT NULL,
		embedding BLOB NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_memories_created_at ON memories(created_at);
	`

	if _, err := s.db.Exec(memoriesSchema); err != nil {
		return fmt.Errorf("failed to create memories table: %w", err)
	}

	// Create history table for conversation turns with FTS5 index
	historySchema := `
	CREATE TABLE IF NOT EXISTS history (
		id TEXT PRIMARY KEY,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		session_id TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_history_created_at ON history(created_at);
	CREATE INDEX IF NOT EXISTS idx_history_session ON history(session_id);
	`

	if _, err := s.db.Exec(historySchema); err != nil {
		return fmt.Errorf("failed to create history table: %w", err)
	}

	// Create FTS5 virtual table for full-text search on history
	historyFTSSchema := `
	CREATE VIRTUAL TABLE IF NOT EXISTS history_fts USING fts5(
		content,
		content='history',
		content_rowid='rowid'
	);

	-- Triggers to keep FTS index in sync with history table
	CREATE TRIGGER IF NOT EXISTS history_ai AFTER INSERT ON history BEGIN
		INSERT INTO history_fts(rowid, content) VALUES (NEW.rowid, NEW.content);
	END;

	CREATE TRIGGER IF NOT EXISTS history_ad AFTER DELETE ON history BEGIN
		INSERT INTO history_fts(history_fts, rowid, content) VALUES('delete', OLD.rowid, OLD.content);
	END;

	CREATE TRIGGER IF NOT EXISTS history_au AFTER UPDATE ON history BEGIN
		INSERT INTO history_fts(history_fts, rowid, content) VALUES('delete', OLD.rowid, OLD.content);
		INSERT INTO history_fts(rowid, content) VALUES (NEW.rowid, NEW.content);
	END;
	`

	if _, err := s.db.Exec(historyFTSSchema); err != nil {
		return fmt.Errorf("failed to create history FTS index: %w", err)
	}

	// Create FTS5 virtual table for full-text search on memories
	memoryFTSSchema := `
	CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
		text,
		content='memories',
		content_rowid='rowid'
	);

	-- Triggers to keep FTS index in sync with memories table
	CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
		INSERT INTO memories_fts(rowid, text) VALUES (NEW.rowid, NEW.text);
	END;

	CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
		INSERT INTO memories_fts(memories_fts, rowid, text) VALUES('delete', OLD.rowid, OLD.text);
	END;

	CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
		INSERT INTO memories_fts(memories_fts, rowid, text) VALUES('delete', OLD.rowid, OLD.text);
		INSERT INTO memories_fts(rowid, text) VALUES (NEW.rowid, NEW.text);
	END;
	`

	if _, err := s.db.Exec(memoryFTSSchema); err != nil {
		return fmt.Errorf("failed to create memories FTS index: %w", err)
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

	_, err = s.db.Exec(`
		INSERT INTO memories (id, text, tags, source, confidence, created_at, provider, model_id, dim, embedding)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.Text, string(tagsJSON), string(item.Source), item.Confidence,
		item.CreatedAt.Unix(), item.Provider, item.ModelID, item.Dim, embeddingBytes)

	if err != nil {
		return fmt.Errorf("failed to save memory: %w", err)
	}

	return nil
}

// GetAllMemories returns all memory items (for vector search).
func (s *Store) GetAllMemories() ([]MemoryItem, error) {
	rows, err := s.db.Query(`
		SELECT id, text, tags, source, confidence, created_at, provider, model_id, dim, embedding
		FROM memories
		ORDER BY created_at DESC
	`)
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
	_, err := s.db.Exec("DELETE FROM memories WHERE id = ?", id)
	return err
}

// SearchMemoriesFTS performs full-text search on memory text.
// Returns top K results ordered by FTS rank.
func (s *Store) SearchMemoriesFTS(query string, topK int) ([]MemoryFTSResult, error) {
	rows, err := s.db.Query(`
		SELECT m.id, m.text, m.tags, m.source, m.confidence, m.created_at,
		       m.provider, m.model_id, m.dim, m.embedding,
		       snippet(memories_fts, 0, '>>>', '<<<', '...', 32) as snippet,
		       rank
		FROM memories m
		JOIN memories_fts fts ON m.rowid = fts.rowid
		WHERE memories_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, query, topK)
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

	_, err := s.db.Exec(`
		INSERT INTO history (id, role, content, created_at, session_id)
		VALUES (?, ?, ?, ?, ?)
	`, item.ID, item.Role, item.Content, item.CreatedAt.Unix(), item.SessionID)

	if err != nil {
		return fmt.Errorf("failed to save history: %w", err)
	}

	return nil
}

// SearchHistory performs full-text search on history content.
// Returns top K results ordered by FTS rank.
func (s *Store) SearchHistory(query string, topK int) ([]HistorySearchResult, error) {
	rows, err := s.db.Query(`
		SELECT h.id, h.role, h.content, h.created_at, h.session_id,
		       snippet(history_fts, 0, '>>>', '<<<', '...', 32) as snippet,
		       rank
		FROM history h
		JOIN history_fts fts ON h.rowid = fts.rowid
		WHERE history_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, query, topK)
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
	rows, err := s.db.Query(`
		SELECT id, role, content, created_at, session_id
		FROM history
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
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
	_, err := s.db.Exec("DELETE FROM history")
	return err
}

// ClearMemories deletes all memory items.
func (s *Store) ClearMemories() error {
	_, err := s.db.Exec("DELETE FROM memories")
	return err
}

