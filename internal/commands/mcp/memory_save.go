package mcp

import (
	"context"
	"fmt"
	"strings"

	memoryservice "github.com/austiecodes/gomor/internal/memory/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MemorySaveInput defines the input schema for the memory save tool
type MemorySaveInput struct {
	Text string `json:"text" jsonschema:"the preference or fact to save"`
	Tags string `json:"tags,omitempty" jsonschema:"comma-separated tags for categorization"`
}

// MemorySaveOutput defines the output schema for the memory save tool
type MemorySaveOutput struct {
	Message string `json:"message" jsonschema:"success message with memory ID"`
	ID      string `json:"id" jsonschema:"the ID of the saved memory"`
}

// handleMemorySave handles the memory_save tool call
func handleMemorySave(ctx context.Context, request *mcp.CallToolRequest, input MemorySaveInput) (*mcp.CallToolResult, MemorySaveOutput, error) {
	// Validate text (required)
	text := strings.TrimSpace(input.Text)
	if text == "" {
		return nil, MemorySaveOutput{}, fmt.Errorf("parameter 'text' must be a non-empty string")
	}

	// Extract tags (optional)
	var tags []string
	if input.Tags != "" {
		for _, t := range strings.Split(input.Tags, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	result, err := memoryservice.Save(ctx, memoryservice.SaveInput{
		Text: text,
		Tags: tags,
	})
	if err != nil {
		return nil, MemorySaveOutput{}, err
	}

	return nil, MemorySaveOutput{
		Message: fmt.Sprintf("Memory saved successfully (id: %s)", result.Item.ID),
		ID:      result.Item.ID,
	}, nil
}
