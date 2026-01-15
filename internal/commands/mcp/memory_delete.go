package mcp

import (
	"context"

	memoryservice "github.com/austiecodes/gomor/internal/memory/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type MemoryDeleteInput struct {
	ID string `json:"id" jsonschema:"the memory id to delete"`
}

type MemoryDeleteOutput struct {
	Message string `json:"message" jsonschema:"delete result message"`
	ID      string `json:"id" jsonschema:"the memory id"`
	Deleted bool   `json:"deleted" jsonschema:"whether a memory row was deleted"`
}

func handleMemoryDelete(ctx context.Context, request *mcp.CallToolRequest, input MemoryDeleteInput) (*mcp.CallToolResult, MemoryDeleteOutput, error) {
	_ = request

	result, err := memoryservice.Delete(ctx, memoryservice.DeleteInput{ID: input.ID})
	if err != nil {
		return nil, MemoryDeleteOutput{}, err
	}

	message := "Memory deleted successfully."
	if !result.Deleted {
		message = "Memory not found."
	}

	return nil, MemoryDeleteOutput{
		Message: message,
		ID:      result.ID,
		Deleted: result.Deleted,
	}, nil
}
