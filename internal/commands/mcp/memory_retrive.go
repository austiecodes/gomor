package mcp

import (
	"context"
	"fmt"
	"strings"

	memoryservice "github.com/austiecodes/gomor/internal/memory/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MemoryRetrieveInput defines the input schema for the memory retrieve tool
type MemoryRetrieveInput struct {
	Query string `json:"query" jsonschema:"the query to search for related memories"`
}

// MemoryRetrieveOutput defines the output schema for the memory retrieve tool
type MemoryRetrieveOutput struct {
	Results string `json:"results" jsonschema:"formatted text containing retrieved memories"`
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
	}, nil
}
