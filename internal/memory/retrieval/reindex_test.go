package retrieval

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/austiecodes/gomor/internal/memory/store"
	"github.com/austiecodes/gomor/internal/provider"
	"github.com/austiecodes/gomor/internal/utils"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

func setupTestStore(t *testing.T) *store.Store {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	s, err := store.NewStoreWithDB(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	return s
}

// TestReindexMemories_Integration uses REAL configuration and API.
// Run with: go test -v ./internal/memory/retrieval -run TestReindexMemories_Integration
func TestReindexMemories_Integration(t *testing.T) {
	// 1. Load real config
	cfg, err := utils.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// 2. Setup real provider
	// We use the configured EmbeddingModel
	embeddingModel := cfg.Model.EmbeddingModel
	if embeddingModel == nil {
		t.Skip("skipping integration test: no embedding model configured")
	}

	t.Logf("Using embedding model: %s / %s", embeddingModel.Provider, embeddingModel.ModelID)

	client, err := provider.NewEmbeddingClient(cfg, embeddingModel.Provider)
	if err != nil {
		t.Fatalf("failed to create embedding client: %v", err)
	}

	// 3. Setup temporary store so we don't mess up user's DB
	// We use an in-memory DB but with real schema
	storeInstance := setupTestStore(t)
	defer storeInstance.Close()

	// 4. Seed test data
	testMemories := []string{
		"Golang is a statically typed, compiled programming language designed at Google.",
		"Concurrency in Go is a first-class citizen, utilizing goroutines and channels.",
		"Interfaces in Go are satisfied implicitly.",
	}

	ids := make([]string, len(testMemories))

	for i, text := range testMemories {
		item := &store.MemoryItem{
			ID:        uuid.New().String(),
			Text:      text,
			Source:    store.SourceExplicit,
			CreatedAt: time.Now(),
			// Initial dummy embedding (to prove it changes)
			Embedding:  []float32{0.0, 0.0},
			Confidence: 1.0,
			Provider:   "dummy",
			ModelID:    "dummy",
			Dim:        2,
		}
		if err := storeInstance.SaveMemory(item); err != nil {
			t.Fatalf("failed to save seed memory: %v", err)
		}
		ids[i] = item.ID
	}

	// 5. Run Reindex
	t.Log("Starting reindex...")
	err = ReindexMemories(context.Background(), storeInstance, client, *embeddingModel)
	if err != nil {
		t.Fatalf("ReindexMemories failed: %v", err)
	}

	// 6. Verify
	memories, _ := storeInstance.GetAllMemories()
	for _, m := range memories {
		// print info
		t.Logf("Memory %s: Provider=%s Model=%s Dim=%d", m.ID, m.Provider, m.ModelID, m.Dim)

		if m.Provider != embeddingModel.Provider {
			t.Errorf("expected provider %s, got %s", embeddingModel.Provider, m.Provider)
		}
		if m.ModelID != embeddingModel.ModelID {
			t.Errorf("expected model %s, got %s", embeddingModel.ModelID, m.ModelID)
		}

		realDim := client.Dimensions(*embeddingModel)
		if m.Dim != realDim {
			t.Errorf("expected dimension %d, got %d", realDim, m.Dim)
		}

		// Check that embedding is not the dummy one
		isZero := true
		for _, v := range m.Embedding {
			if v != 0 {
				isZero = false
				break
			}
		}
		if isZero {
			t.Error("embedding is still zero-vector")
		}
	}
}
