package retrieval

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/austiecodes/goa/internal/client"
	"github.com/austiecodes/goa/internal/provider"
	"github.com/austiecodes/goa/internal/types"
	"github.com/austiecodes/goa/internal/utils"
)

// fakeEmbeddingClient returns deterministic vectors based on input text.
type fakeEmbeddingClient struct{}

func (f *fakeEmbeddingClient) Embed(ctx context.Context, model types.Model, text string) ([]float32, error) {
	// Very simple routing: if the text contains key terms, return a vector that will match.
	if containsAny(text, []string{"virtual", "polymorphism", "inheritance", "C++"}) {
		return []float32{1, 0}, nil
	}
	return []float32{0, 1}, nil
}

func (f *fakeEmbeddingClient) EmbedBatch(ctx context.Context, model types.Model, texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))
	for i, t := range texts {
		v, _ := f.Embed(ctx, model, t)
		vectors[i] = v
	}
	return vectors, nil
}

func (f *fakeEmbeddingClient) Dimensions(model types.Model) int {
	return 2
}

// fakeStream implements client.StreamResponse for canned responses.
type fakeStream struct {
	chunks []string
	idx    int
}

func (s *fakeStream) Next() bool {
	if s.idx < len(s.chunks) {
		s.idx++
		return true
	}
	return false
}

func (s *fakeStream) GetChunk() string {
	if s.idx == 0 || s.idx > len(s.chunks) {
		return ""
	}
	return s.chunks[s.idx-1]
}

func (s *fakeStream) Err() error   { return nil }
func (s *fakeStream) Close() error { return nil }

// fakeQueryClient returns fixed ANSWER / REPHRASE output.
type fakeQueryClient struct{}

func (f *fakeQueryClient) ChatStream(ctx context.Context, model types.Model, query string) (client.StreamResponse, error) {
	text := "ANSWER: C++ virtual functions enable polymorphism\nREPHRASE: C++ virtual functions polymorphism inheritance"
	return &fakeStream{chunks: []string{text}}, nil
}

func (f *fakeQueryClient) ChatStreamWithContext(ctx context.Context, model types.Model, systemContext, query string) (client.StreamResponse, error) {
	// ignore systemContext for the fake
	return f.ChatStream(ctx, model, query)
}

func (f *fakeQueryClient) ListModels(ctx context.Context) ([]string, error) {
	return []string{"fake-model"}, nil
}

// containsAny reports whether text contains any of the needles (case-insensitive).
func containsAny(text string, needles []string) bool {
	lower := strings.ToLower(text)
	for _, n := range needles {
		if strings.Contains(lower, strings.ToLower(n)) {
			return true
		}
	}
	return false
}

// TestRetriever_EndToEnd_FakeClients exercises the retrieval pipeline with fake clients and real store/config.
func TestRetriever_EndToEnd_FakeClients(t *testing.T) {
	ctx := context.Background()

	cfg, err := utils.LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	// Relax thresholds for the test
	cfg.Memory.MinSimilarity = 0.1
	cfg.Memory.MemoryTopK = 10
	cfg.Memory.FTSStrategy = utils.FTSStrategyDirect

	// Ensure models are not nil (defaults are applied in LoadConfig, but guard anyway)
	if cfg.Model.EmbeddingModel == nil {
		cfg.Model.EmbeddingModel = &types.Model{Provider: "fake", ModelID: "fake-embed"}
	}
	if cfg.Model.ToolModel == nil {
		cfg.Model.ToolModel = &types.Model{Provider: "fake", ModelID: "fake-tool"}
	}

	store, err := NewStore()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	// Insert a test memory
	memText := "C++ virtual functions enable polymorphism via inheritance"
	embClient := &fakeEmbeddingClient{}
	vec, _ := embClient.Embed(ctx, *cfg.Model.EmbeddingModel, memText)
	vec = NormalizeVector(vec)

	item := &MemoryItem{
		Text:       memText,
		Tags:       []string{"c++", "polymorphism"},
		Source:     SourceExplicit,
		Confidence: 1.0,
		Provider:   cfg.Model.EmbeddingModel.Provider,
		ModelID:    cfg.Model.EmbeddingModel.ModelID,
		Dim:        len(vec),
		Embedding:  vec,
		CreatedAt:  time.Now(),
	}
	if err := store.SaveMemory(item); err != nil {
		t.Fatalf("save memory: %v", err)
	}
	// Cleanup after test
	defer func() {
		_ = store.DeleteMemory(item.ID)
	}()

	// Build retriever with fake clients
	retriever := NewRetriever(
		store,
		embClient,
		&fakeQueryClient{},
		*cfg.Model.EmbeddingModel,
		*cfg.Model.ToolModel,
		cfg.Memory,
	)

	query := "C++ virtual functions polymorphism inheritance"
	resp, err := retriever.Retrieve(ctx, query)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}

	if len(resp.Results) == 0 {
		t.Fatalf("expected results, got 0")
	}

	// Log the unified results for debugging
	for i, r := range resp.Results {
		t.Logf("%d) score=%.2f source=%s text=%s snippet=%s", i+1, r.Score, r.Source, r.Item.Text, r.Snippet)
	}

	// Basic assertion: top result should be the inserted memory
	if resp.Results[0].Item.ID != item.ID {
		t.Fatalf("top result mismatch: got %s want %s", resp.Results[0].Item.ID, item.ID)
	}
}

// TestRetriever_RealClients_Debug uses real embedding/query clients from config
// and prints all intermediate results for debugging.
// Run with: go test ./internal/memory -run TestRetriever_RealClients_Debug -v
func TestRetriever_RealClients_Debug(t *testing.T) {
	ctx := context.Background()

	// Load real config
	cfg, err := utils.LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	fmt.Println("========== CONFIG ==========")
	fmt.Printf("EmbeddingModel: %+v\n", cfg.Model.EmbeddingModel)
	fmt.Printf("ToolModel: %+v\n", cfg.Model.ToolModel)
	fmt.Printf("Memory Config: %+v\n", cfg.Memory)
	fmt.Println()

	if cfg.Model.EmbeddingModel == nil {
		t.Fatal("embedding_model not configured. Run 'goa set' first.")
	}

	// Open real store
	store, err := NewStore()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	// List existing memories
	fmt.Println("========== EXISTING MEMORIES ==========")
	memories, err := store.GetAllMemories()
	if err != nil {
		t.Fatalf("get all memories: %v", err)
	}
	if len(memories) == 0 {
		fmt.Println("No memories in store. Add some with 'goa memory' or goa_memory_save first.")
	} else {
		for i, m := range memories {
			fmt.Printf("%d) ID=%s Text=%s Tags=%v\n", i+1, m.ID[:8], m.Text, m.Tags)
		}
	}
	fmt.Println()

	// Create real embedding client
	embeddingModel := *cfg.Model.EmbeddingModel
	embClient, err := provider.NewEmbeddingClient(cfg, embeddingModel.Provider)
	if err != nil {
		t.Fatalf("create embedding client: %v", err)
	}

	// Create real query client (may be nil if tool_model not configured)
	var queryClient client.QueryClient
	if cfg.Model.ToolModel != nil {
		toolModel := *cfg.Model.ToolModel
		queryClient, _ = provider.NewQueryClient(cfg, toolModel.Provider)
	}

	// Build retriever
	var toolModel types.Model
	if cfg.Model.ToolModel != nil {
		toolModel = *cfg.Model.ToolModel
	}
	retriever := NewRetriever(
		store,
		embClient,
		queryClient,
		embeddingModel,
		toolModel,
		cfg.Memory,
	)

	// Insert a test memory that should match the query
	fmt.Println("========== INSERTING TEST MEMORY ==========")
	testMemText := "C++ virtual functions enable polymorphism through inheritance hierarchies"
	testVec, err := embClient.Embed(ctx, embeddingModel, testMemText)
	if err != nil {
		t.Fatalf("embed test memory: %v", err)
	}
	testVec = NormalizeVector(testVec)
	testItem := &MemoryItem{
		Text:       testMemText,
		Tags:       []string{"cpp", "oop", "test"},
		Source:     SourceExplicit,
		Confidence: 1.0,
		Provider:   embeddingModel.Provider,
		ModelID:    embeddingModel.ModelID,
		Dim:        len(testVec),
		Embedding:  testVec,
		CreatedAt:  time.Now(),
	}
	if err := store.SaveMemory(testItem); err != nil {
		t.Fatalf("save test memory: %v", err)
	}
	fmt.Printf("Inserted test memory: ID=%s Text=%s\n", testItem.ID[:8], testItem.Text)
	// Cleanup after test
	defer func() {
		_ = store.DeleteMemory(testItem.ID)
		fmt.Println("Cleaned up test memory")
	}()
	fmt.Println()

	// Test query
	query := "c++ virtual functions"
	fmt.Println("========== QUERY ==========")
	fmt.Printf("Query: %s\n", query)
	fmt.Println()

	// Step 1: Query transformation
	fmt.Println("========== STEP 1: QUERY TRANSFORMATION ==========")
	transformedQueries, err := retriever.transformQueryForVector(ctx, query)
	if err != nil {
		fmt.Printf("Transform error: %v\n", err)
	} else {
		for i, q := range transformedQueries {
			fmt.Printf("Transformed[%d]: %s\n", i, q)
		}
	}
	fmt.Println()

	// Step 2: Vector search
	fmt.Println("========== STEP 2: VECTOR SEARCH ==========")
	vectorResults, err := retriever.vectorSearch(ctx, query)
	if err != nil {
		fmt.Printf("Vector search error: %v\n", err)
	} else {
		fmt.Printf("Vector results: %d\n", len(vectorResults))
		for i, r := range vectorResults {
			fmt.Printf("  %d) sim=%.4f text=%s\n", i+1, r.Similarity, r.Item.Text)
		}
	}
	fmt.Println()

	// Step 3: FTS search
	fmt.Println("========== STEP 3: FTS SEARCH ==========")
	ftsResults, err := retriever.ftsSearch(ctx, query)
	if err != nil {
		fmt.Printf("FTS search error: %v\n", err)
	} else {
		fmt.Printf("FTS results: %d\n", len(ftsResults))
		for i, r := range ftsResults {
			fmt.Printf("  %d) rank=%.4f text=%s snippet=%s\n", i+1, r.Rank, r.Item.Text, r.Snippet)
		}
	}
	fmt.Println()

	// Step 4: Fusion
	fmt.Println("========== STEP 4: FUSION ==========")
	resp, err := retriever.Retrieve(ctx, query)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	fmt.Printf("Unified results: %d\n", len(resp.Results))
	for i, r := range resp.Results {
		fmt.Printf("  %d) score=%.4f source=%s vectorScore=%.4f ftsRank=%.4f text=%s\n",
			i+1, r.Score, r.Source, r.VectorScore, r.FTSRank, r.Item.Text)
	}
	fmt.Println()

	// Final output
	fmt.Println("========== FINAL OUTPUT ==========")
	fmt.Println(FormatAsText(resp))
}
