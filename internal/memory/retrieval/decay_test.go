package retrieval

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/austiecodes/gomor/internal/memory/store"
	"github.com/austiecodes/gomor/internal/types"
	"github.com/austiecodes/gomor/internal/utils"
	_ "modernc.org/sqlite"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	memStore, err := store.NewStoreWithDB(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	t.Cleanup(func() {
		_ = memStore.Close()
	})

	return memStore
}

func newTestRetriever(memStore *store.Store) *Retriever {
	config := utils.DefaultConfig()
	config.Memory.MinSimilarity = 0.1
	config.Memory.MemoryTopK = 10

	return NewRetriever(
		memStore,
		&fakeEmbeddingClient{},
		nil,
		types.Model{Provider: "fake", ModelID: "fake-embedding"},
		types.Model{},
		config.Memory,
	)
}

func TestRetrievePrefersFresherMemory(t *testing.T) {
	memStore := newTestStore(t)
	retriever := newTestRetriever(memStore)
	now := time.Now().UTC()

	older := &MemoryItem{
		Text:      "C++ virtual functions enable polymorphism",
		Source:    SourceExplicit,
		CreatedAt: now.Add(-60 * 24 * time.Hour),
		Provider:  "fake",
		ModelID:   "fake-embedding",
		Dim:       2,
		Embedding: NormalizeVector([]float32{1, 0}),
	}
	if err := memStore.SaveMemory(older); err != nil {
		t.Fatalf("save older memory: %v", err)
	}

	fresher := &MemoryItem{
		Text:      "C++ virtual functions enable polymorphism",
		Source:    SourceExplicit,
		CreatedAt: now.Add(-2 * 24 * time.Hour),
		Provider:  "fake",
		ModelID:   "fake-embedding",
		Dim:       2,
		Embedding: NormalizeVector([]float32{1, 0}),
	}
	if err := memStore.SaveMemory(fresher); err != nil {
		t.Fatalf("save fresher memory: %v", err)
	}

	resp, err := retriever.Retrieve(context.Background(), "C++ virtual functions polymorphism")
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(resp.Results) < 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}
	if resp.Results[0].Item.ID != fresher.ID {
		t.Fatalf("expected fresher memory first, got %s want %s", resp.Results[0].Item.ID, fresher.ID)
	}
	if resp.Results[0].Freshness <= resp.Results[1].Freshness {
		t.Fatalf("expected fresher memory to have higher freshness, got %.4f <= %.4f", resp.Results[0].Freshness, resp.Results[1].Freshness)
	}
}

func TestRetrieveWeaklyReinforcesTopResult(t *testing.T) {
	memStore := newTestStore(t)
	retriever := newTestRetriever(memStore)
	now := time.Now().UTC()

	item := &MemoryItem{
		Text:      "C++ virtual functions enable polymorphism",
		Source:    SourceExplicit,
		CreatedAt: now.Add(-90 * 24 * time.Hour),
		Provider:  "fake",
		ModelID:   "fake-embedding",
		Dim:       2,
		Embedding: NormalizeVector([]float32{1, 0}),
	}
	if err := memStore.SaveMemory(item); err != nil {
		t.Fatalf("save memory: %v", err)
	}

	beforeConfidence := item.Confidence
	beforeStability := item.StabilityDays

	resp, err := retriever.Retrieve(context.Background(), "C++ virtual functions polymorphism")
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}

	memories, err := memStore.GetAllMemories()
	if err != nil {
		t.Fatalf("get all memories: %v", err)
	}
	if len(memories) != 1 {
		t.Fatalf("expected 1 stored memory, got %d", len(memories))
	}

	stored := memories[0]
	if stored.LastRetrievedAt == nil {
		t.Fatal("expected retrieval to update last_retrieved_at")
	}
	if stored.StabilityDays <= beforeStability {
		t.Fatalf("expected stability to increase, got %.2f <= %.2f", stored.StabilityDays, beforeStability)
	}
	if stored.Confidence != beforeConfidence {
		t.Fatalf("expected confidence to remain unchanged, got %.2f want %.2f", stored.Confidence, beforeConfidence)
	}
}
