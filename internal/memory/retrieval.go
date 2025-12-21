package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/austiecodes/goa/internal/client"
	"github.com/austiecodes/goa/internal/types"
	"github.com/austiecodes/goa/internal/utils"
)

// Retriever performs hybrid retrieval from memory using vector search and FTS.
type Retriever struct {
	store           *Store
	embeddingClient client.EmbeddingClient
	queryClient     client.QueryClient
	embeddingModel  types.Model
	toolModel       types.Model
	config          utils.MemoryConfig
}

// NewRetriever creates a new retriever with the given dependencies.
func NewRetriever(
	store *Store,
	embeddingClient client.EmbeddingClient,
	queryClient client.QueryClient,
	embeddingModel types.Model,
	toolModel types.Model,
	config utils.MemoryConfig,
) *Retriever {
	return &Retriever{
		store:           store,
		embeddingClient: embeddingClient,
		queryClient:     queryClient,
		embeddingModel:  embeddingModel,
		toolModel:       toolModel,
		config:          config,
	}
}

// Retrieve performs unified memory retrieval using both vector search and FTS.
// 1. Uses tool_model to transform the query (answer + rephrase)
// 2. Embeds transformed queries and performs vector search
// 3. Performs FTS based on configured strategy
// 4. Fuses and ranks results
func (r *Retriever) Retrieve(ctx context.Context, query string) (*RetrievalResponse, error) {
	var (
		vectorResults []SearchResult
		ftsResults    []MemoryFTSResult
		vectorErr     error
		ftsErr        error
		wg            sync.WaitGroup
	)

	// Run vector search path in parallel
	wg.Add(1)
	go func() {
		defer wg.Done()
		vectorResults, vectorErr = r.vectorSearch(ctx, query)
	}()

	// Run FTS search path in parallel
	wg.Add(1)
	go func() {
		defer wg.Done()
		ftsResults, ftsErr = r.ftsSearch(ctx, query)
	}()

	wg.Wait()

	// Log errors but continue if at least one path succeeded
	if vectorErr != nil && ftsErr != nil {
		return nil, fmt.Errorf("retrieval failed: vector: %v, fts: %v", vectorErr, ftsErr)
	}

	// Fuse results
	unified := r.fuseResults(vectorResults, ftsResults)

	return &RetrievalResponse{
		Results: unified,
		Query:   query,
	}, nil
}

// vectorSearch performs vector similarity search with LLM query transformation.
func (r *Retriever) vectorSearch(ctx context.Context, query string) ([]SearchResult, error) {
	// Transform query using tool_model: get brief answer and rephrased query
	transformedQueries, err := r.transformQueryForVector(ctx, query)
	if err != nil {
		// Fallback to original query if transformation fails
		transformedQueries = []string{query}
	}

	// Embed all transformed queries and collect results
	var allResults []SearchResult
	seenIDs := make(map[string]bool)

	for _, q := range transformedQueries {
		embedding, err := r.embeddingClient.Embed(ctx, r.embeddingModel, q)
		if err != nil {
			continue // skip failed embeddings
		}

		results, err := r.store.SearchMemories(embedding, r.config.MemoryTopK, r.config.MinSimilarity)
		if err != nil {
			continue
		}

		// Deduplicate
		for _, res := range results {
			if !seenIDs[res.Item.ID] {
				seenIDs[res.Item.ID] = true
				allResults = append(allResults, res)
			}
		}
	}

	// Re-sort by similarity and limit
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Similarity > allResults[j].Similarity
	})

	if len(allResults) > r.config.MemoryTopK {
		allResults = allResults[:r.config.MemoryTopK]
	}

	return allResults, nil
}

// transformQueryForVector uses tool_model to generate transformed queries for better embedding.
// Returns: [brief answer, rephrased query for search]
func (r *Retriever) transformQueryForVector(ctx context.Context, query string) ([]string, error) {
	if r.queryClient == nil {
		return []string{query}, nil
	}

	prompt := fmt.Sprintf(`Given this user query, provide two transformations for memory retrieval:
1. A brief 1-2 sentence answer to the query (as if you know the answer)
2. A rephrased version optimized for semantic search

User query: %s

Respond in this exact format (no other text):
ANSWER: <brief answer>
REPHRASE: <rephrased query>`, query)

	stream, err := r.queryClient.ChatStream(ctx, r.toolModel, prompt)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	var sb strings.Builder
	for stream.Next() {
		sb.WriteString(stream.GetChunk())
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}

	response := sb.String()
	return parseTransformResponse(response, query), nil
}

// parseTransformResponse extracts transformed queries from LLM response.
func parseTransformResponse(response, originalQuery string) []string {
	results := []string{originalQuery} // always include original

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ANSWER:") {
			answer := strings.TrimSpace(strings.TrimPrefix(line, "ANSWER:"))
			if answer != "" {
				results = append(results, answer)
			}
		} else if strings.HasPrefix(line, "REPHRASE:") {
			rephrase := strings.TrimSpace(strings.TrimPrefix(line, "REPHRASE:"))
			if rephrase != "" {
				results = append(results, rephrase)
			}
		}
	}

	return results
}

// ftsSearch performs FTS based on the configured strategy.
func (r *Retriever) ftsSearch(ctx context.Context, query string) ([]MemoryFTSResult, error) {
	strategy := r.config.FTSStrategy
	if strategy == "" {
		strategy = utils.FTSStrategyDirect
	}

	switch strategy {
	case utils.FTSStrategyDirect:
		return r.ftsSearchDirect(query)
	case utils.FTSStrategySummary:
		return r.ftsSearchSummary(ctx, query)
	case utils.FTSStrategyKeywords:
		return r.ftsSearchKeywords(ctx, query)
	case utils.FTSStrategyAuto:
		return r.ftsSearchAuto(ctx, query)
	default:
		return r.ftsSearchDirect(query)
	}
}

// ftsSearchDirect tokenizes the raw query and performs FTS.
func (r *Retriever) ftsSearchDirect(query string) ([]MemoryFTSResult, error) {
	ftsQuery := tokenizeForFTS(query)
	if ftsQuery == "" {
		return nil, nil
	}
	return r.store.SearchMemoriesFTS(ftsQuery, r.config.MemoryTopK)
}

// ftsSearchSummary uses tool_model to summarize the query, then performs FTS.
func (r *Retriever) ftsSearchSummary(ctx context.Context, query string) ([]MemoryFTSResult, error) {
	if r.queryClient == nil {
		return r.ftsSearchDirect(query)
	}

	prompt := fmt.Sprintf(`Summarize this query in one short sentence for text search:
Query: %s

Respond with ONLY the summary, no other text.`, query)

	stream, err := r.queryClient.ChatStream(ctx, r.toolModel, prompt)
	if err != nil {
		return r.ftsSearchDirect(query) // fallback
	}
	defer stream.Close()

	var sb strings.Builder
	for stream.Next() {
		sb.WriteString(stream.GetChunk())
	}

	summary := strings.TrimSpace(sb.String())
	if summary == "" {
		return r.ftsSearchDirect(query)
	}

	ftsQuery := tokenizeForFTS(summary)
	if ftsQuery == "" {
		return nil, nil
	}
	return r.store.SearchMemoriesFTS(ftsQuery, r.config.MemoryTopK)
}

// ftsSearchKeywords uses tool_model to extract keywords, then performs FTS.
func (r *Retriever) ftsSearchKeywords(ctx context.Context, query string) ([]MemoryFTSResult, error) {
	if r.queryClient == nil {
		return r.ftsSearchDirect(query)
	}

	prompt := fmt.Sprintf(`Extract 3-5 key search terms from this query:
Query: %s

Respond with ONLY comma-separated keywords, no other text.`, query)

	stream, err := r.queryClient.ChatStream(ctx, r.toolModel, prompt)
	if err != nil {
		return r.ftsSearchDirect(query) // fallback
	}
	defer stream.Close()

	var sb strings.Builder
	for stream.Next() {
		sb.WriteString(stream.GetChunk())
	}

	keywords := strings.TrimSpace(sb.String())
	if keywords == "" {
		return r.ftsSearchDirect(query)
	}

	// Parse keywords and build FTS query
	parts := strings.Split(keywords, ",")
	var terms []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.ReplaceAll(p, "\"", "")
		if p != "" {
			terms = append(terms, p)
		}
	}

	if len(terms) == 0 {
		return r.ftsSearchDirect(query)
	}

	ftsQuery := strings.Join(terms, " OR ")
	return r.store.SearchMemoriesFTS(ftsQuery, r.config.MemoryTopK)
}

// ftsSearchAuto tries direct first, falls back to summary if few results.
func (r *Retriever) ftsSearchAuto(ctx context.Context, query string) ([]MemoryFTSResult, error) {
	results, err := r.ftsSearchDirect(query)
	if err != nil {
		return nil, err
	}

	// If we got enough results, return them
	threshold := r.config.MemoryTopK / 2
	if threshold < 3 {
		threshold = 3
	}
	if len(results) >= threshold {
		return results, nil
	}

	// Otherwise, try summary-based search
	summaryResults, err := r.ftsSearchSummary(ctx, query)
	if err != nil {
		return results, nil // return what we have
	}

	// Merge and deduplicate
	seenIDs := make(map[string]bool)
	for _, r := range results {
		seenIDs[r.Item.ID] = true
	}
	for _, r := range summaryResults {
		if !seenIDs[r.Item.ID] {
			seenIDs[r.Item.ID] = true
			results = append(results, r)
		}
	}

	return results, nil
}

// tokenizeForFTS converts a query string to an FTS-safe query.
func tokenizeForFTS(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}

	// Split into words and filter
	words := strings.Fields(query)
	var tokens []string
	for _, w := range words {
		// Remove FTS special characters (FTS5 operators: AND OR NOT NEAR + - * ^ : " ')
		w = strings.ReplaceAll(w, "\"", "")
		w = strings.ReplaceAll(w, "'", "")
		w = strings.ReplaceAll(w, "*", "")
		w = strings.ReplaceAll(w, "-", " ")
		w = strings.ReplaceAll(w, "+", "")
		w = strings.ReplaceAll(w, "^", "")
		w = strings.ReplaceAll(w, ":", "")
		w = strings.ReplaceAll(w, "(", "")
		w = strings.ReplaceAll(w, ")", "")
		w = strings.TrimSpace(w)
		if len(w) > 1 { // skip single characters
			tokens = append(tokens, w)
		}
	}

	if len(tokens) == 0 {
		return ""
	}

	// Join with OR for broader matching
	return strings.Join(tokens, " OR ")
}

// fuseResults combines vector and FTS results into a unified ranked list.
func (r *Retriever) fuseResults(vectorResults []SearchResult, ftsResults []MemoryFTSResult) []UnifiedResult {
	// Build a map of results by ID
	resultMap := make(map[string]*UnifiedResult)

	// Add vector results
	for _, vr := range vectorResults {
		resultMap[vr.Item.ID] = &UnifiedResult{
			Item:        vr.Item,
			VectorScore: vr.Similarity,
			Source:      "vector",
		}
	}

	// Add/merge FTS results
	for _, fr := range ftsResults {
		if existing, ok := resultMap[fr.Item.ID]; ok {
			// Memory found in both - mark as "both"
			existing.Source = "both"
			existing.FTSRank = fr.Rank
			existing.Snippet = fr.Snippet
		} else {
			resultMap[fr.Item.ID] = &UnifiedResult{
				Item:    fr.Item,
				FTSRank: fr.Rank,
				Snippet: fr.Snippet,
				Source:  "fts",
			}
		}
	}

	// Calculate unified scores and convert to slice
	var results []UnifiedResult
	for _, ur := range resultMap {
		ur.Score = calculateUnifiedScore(ur)
		results = append(results, *ur)
	}

	// Sort by unified score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit to top K
	if len(results) > r.config.MemoryTopK {
		results = results[:r.config.MemoryTopK]
	}

	return results
}

// calculateUnifiedScore computes a normalized score for ranking.
// Memories found in both vector and FTS get a boost.
func calculateUnifiedScore(ur *UnifiedResult) float64 {
	var score float64

	switch ur.Source {
	case "vector":
		// Vector similarity is already 0-1
		score = ur.VectorScore
	case "fts":
		// FTS rank is negative (lower is better), normalize to 0-1
		// Typical ranks are -10 to 0, so we map that range
		score = 1.0 + (ur.FTSRank / 20.0) // maps -20 to 0, 0 to 1
		if score < 0 {
			score = 0
		}
		if score > 1 {
			score = 1
		}
	case "both":
		// Boost for appearing in both
		vectorScore := ur.VectorScore
		ftsScore := 1.0 + (ur.FTSRank / 20.0)
		if ftsScore < 0 {
			ftsScore = 0
		}
		if ftsScore > 1 {
			ftsScore = 1
		}
		// Weighted combination with boost
		score = (vectorScore*0.6 + ftsScore*0.4) * 1.2
		if score > 1 {
			score = 1
		}
	}

	return score
}

// FormatAsText formats the retrieval results as readable text.
func FormatAsText(resp *RetrievalResponse) string {
	if resp == nil || len(resp.Results) == 0 {
		return "No memories found."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d memories:\n\n", len(resp.Results)))

	for i, r := range resp.Results {
		sb.WriteString(fmt.Sprintf("%d. [%.2f] %s\n", i+1, r.Score, r.Item.Text))
		if len(r.Item.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("   Tags: %s\n", strings.Join(r.Item.Tags, ", ")))
		}
		sb.WriteString(fmt.Sprintf("   Source: %s\n", r.Source))
	}

	return sb.String()
}
