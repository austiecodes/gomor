package memtypes

import "time"

// MemorySource indicates how a memory was created.
type MemorySource string

const (
	// SourceExplicit means the memory was explicitly saved by the user or model.
	SourceExplicit MemorySource = "explicit"
	// SourceExtracted means the memory was automatically extracted from conversation.
	SourceExtracted MemorySource = "extracted"
)

// MemoryItem represents a single preference/fact stored in memory.
type MemoryItem struct {
	ID         string       `json:"id"`
	Text       string       `json:"text"`
	Tags       []string     `json:"tags,omitempty"`
	Source     MemorySource `json:"source"`
	Confidence float64      `json:"confidence"`
	CreatedAt  time.Time    `json:"created_at"`
	Provider   string       `json:"provider"`
	ModelID    string       `json:"model_id"`
	Dim        int          `json:"dim"`
	Embedding  []float32    `json:"-"` // stored as blob, not JSON
}

// HistoryItem represents a conversation turn stored in history.
type HistoryItem struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	SessionID string    `json:"session_id,omitempty"`
}

// SearchResult represents a memory search result with similarity score (vector search).
type SearchResult struct {
	Item       MemoryItem `json:"item"`
	Similarity float64    `json:"similarity"`
}

// MemoryFTSResult represents a memory search result from FTS.
type MemoryFTSResult struct {
	Item    MemoryItem `json:"item"`
	Snippet string     `json:"snippet"` // matched snippet with context
	Rank    float64    `json:"rank"`    // FTS rank score (lower is better)
}

// HistorySearchResult represents a history search result.
type HistorySearchResult struct {
	Item    HistoryItem `json:"item"`
	Snippet string      `json:"snippet"` // matched snippet with context
	Rank    float64     `json:"rank"`    // FTS rank score
}

// UnifiedResult represents a unified retrieval result from any source.
// Used for fusion and ranking across different retrieval methods.
type UnifiedResult struct {
	Item        MemoryItem `json:"item"`
	Score       float64    `json:"score"`        // normalized score (0-1, higher is better)
	Source      string     `json:"source"`       // "vector", "fts", or "both"
	VectorScore float64    `json:"vector_score"` // original vector similarity
	FTSRank     float64    `json:"fts_rank"`     // original FTS rank
	Snippet     string     `json:"snippet"`      // FTS snippet if available
}

// InjectedContext represents the fused retrieval context to inject into prompts.
type InjectedContext struct {
	MemoryFacts     []SearchResult        `json:"memory_facts,omitempty"`
	HistorySnippets []HistorySearchResult `json:"history_snippets,omitempty"`
}

// RetrievalResponse represents the response from the unified memory retrieve operation.
type RetrievalResponse struct {
	Results []UnifiedResult `json:"results"`
	Query   string          `json:"query"`
}
