package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/austiecodes/gomor/internal/memory/retrieval"
	memoryservice "github.com/austiecodes/gomor/internal/memory/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MemoryRetrieveInput defines the input schema for the memory retrieve tool
type MemoryRetrieveInput struct {
	Query string `json:"query" jsonschema:"the query to search for related memories"`
}

// MemoryRetrieveOutput defines the output schema for the memory retrieve tool
type MemoryRetrieveOutput struct {
	Results string                `json:"results" jsonschema:"formatted text containing retrieved memories"`
	Matches []MemoryRetrieveMatch `json:"matches,omitempty" jsonschema:"structured retrieved memories"`
}

type MemoryRetrieveMatch struct {
	ID     string   `json:"id" jsonschema:"memory id"`
	Text   string   `json:"text" jsonschema:"memory text"`
	Tags   []string `json:"tags,omitempty" jsonschema:"memory tags"`
	Score  float64  `json:"score" jsonschema:"final ranking score"`
	Source string   `json:"source" jsonschema:"retrieval source"`
}

// handleMemoryRetrieve handles the goa_memory_retrieve tool call (unified hybrid search)
func handleMemoryRetrieve(ctx context.Context, request *mcp.CallToolRequest, input MemoryRetrieveInput) (*mcp.CallToolResult, MemoryRetrieveOutput, error) {
	// Validate query (required)
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return nil, MemoryRetrieveOutput{}, fmt.Errorf("parameter 'query' must be a non-empty string")
	}

	result, err := memoryservice.Retrieve(ctx, memoryservice.RetrieveInput{Query: query})
	if err != nil {
		return nil, MemoryRetrieveOutput{}, err
	}
	return nil, MemoryRetrieveOutput{
		Results: result.Text,
		Matches: buildRetrieveMatches(result.Response),
	}, nil
}

func buildRetrieveMatches(resp *retrieval.RetrievalResponse) []MemoryRetrieveMatch {
	if resp == nil {
		return nil
	}

	matches := make([]MemoryRetrieveMatch, 0, len(resp.Results))
	for _, result := range resp.Results {
		matches = append(matches, MemoryRetrieveMatch{
			ID:     result.Item.ID,
			Text:   result.Item.Text,
			Tags:   result.Item.Tags,
			Score:  result.Score,
			Source: result.Source,
		})
	}
	return matches
}
