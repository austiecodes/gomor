package store

import (
	"database/sql"
	"testing"
	"time"

	"github.com/austiecodes/gomor/internal/memory/decay"
	"github.com/austiecodes/gomor/internal/memory/memtypes"
	"github.com/austiecodes/gomor/internal/memory/memutils"
	_ "modernc.org/sqlite"
)

func TestNewStoreWithDB_MigratesDecayColumns(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE memories (
			id TEXT PRIMARY KEY,
			text TEXT NOT NULL,
			tags TEXT,
			source TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			provider TEXT NOT NULL,
			model_id TEXT NOT NULL,
			dim INTEGER NOT NULL,
			embedding BLOB NOT NULL
		);`); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}

	createdAt := time.Now().Add(-48 * time.Hour).Unix()
	if _, err := db.Exec(
		`INSERT INTO memories (id, text, tags, source, created_at, provider, model_id, dim, embedding) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"mem-1", "legacy memory", "[]", string(memtypes.SourceExplicit), createdAt, "openai", "test-model", 2, memutils.VectorToBytes([]float32{1, 0}),
	); err != nil {
		t.Fatalf("insert legacy memory: %v", err)
	}

	memStore, err := NewStoreWithDB(db)
	if err != nil {
		t.Fatalf("new store with db: %v", err)
	}

	memories, err := memStore.GetAllMemories()
	if err != nil {
		t.Fatalf("get all memories: %v", err)
	}
	if len(memories) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(memories))
	}

	memory := memories[0]
	if memory.Confidence != decay.DefaultConfidence(memtypes.SourceExplicit) {
		t.Fatalf("unexpected confidence: got %.2f want %.2f", memory.Confidence, decay.DefaultConfidence(memtypes.SourceExplicit))
	}
	if memory.StabilityDays != decay.DefaultStabilityDays(memtypes.SourceExplicit) {
		t.Fatalf("unexpected stability: got %.2f want %.2f", memory.StabilityDays, decay.DefaultStabilityDays(memtypes.SourceExplicit))
	}
	if memory.LastRetrievedAt != nil {
		t.Fatalf("expected nil last retrieved at for legacy memory, got %v", memory.LastRetrievedAt)
	}
}
