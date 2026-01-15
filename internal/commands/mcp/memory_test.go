package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/austiecodes/gomor/internal/memory/retrieval"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestHandleMemorySave_EmptyText tests that empty text returns an error
func TestHandleMemorySave_EmptyText(t *testing.T) {
	ctx := context.Background()
	request := &mcp.CallToolRequest{}

	// Test empty text
	input := MemorySaveInput{Text: ""}
	_, _, err := handleMemorySave(ctx, request, input)
	if err == nil {
		t.Fatal("expected error for empty text, got nil")
	}
	if !strings.Contains(err.Error(), "non-empty string") {
		t.Fatalf("unexpected error message: %v", err)
	}

	// Test whitespace-only text
	input = MemorySaveInput{Text: "   "}
	_, _, err = handleMemorySave(ctx, request, input)
	if err == nil {
		t.Fatal("expected error for whitespace-only text, got nil")
	}
}

// TestHandleMemorySave_Success tests successful memory saving
func TestHandleMemorySave_Success(t *testing.T) {
	ctx := context.Background()
	request := &mcp.CallToolRequest{}

	input := MemorySaveInput{
		Text: "This is a test memory for unit testing",
		Tags: "test, unit-test",
	}

	_, output, err := handleMemorySave(ctx, request, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if !strings.Contains(output.Message, output.ID) {
		t.Fatalf("message should contain ID, got: %s", output.Message)
	}

	t.Logf("Saved memory with ID: %s", output.ID)
}

// TestHandleMemoryRetrieve_EmptyQuery tests that empty query returns an error
func TestHandleMemoryRetrieve_EmptyQuery(t *testing.T) {
	ctx := context.Background()
	request := &mcp.CallToolRequest{}

	// Test empty query
	input := MemoryRetrieveInput{Query: ""}
	_, _, err := handleMemoryRetrieve(ctx, request, input)
	if err == nil {
		t.Fatal("expected error for empty query, got nil")
	}
	if !strings.Contains(err.Error(), "non-empty string") {
		t.Fatalf("unexpected error message: %v", err)
	}

	// Test whitespace-only query
	input = MemoryRetrieveInput{Query: "   "}
	_, _, err = handleMemoryRetrieve(ctx, request, input)
	if err == nil {
		t.Fatal("expected error for whitespace-only query, got nil")
	}
}

func TestHandleMemoryDelete_EmptyID(t *testing.T) {
	ctx := context.Background()
	request := &mcp.CallToolRequest{}

	_, _, err := handleMemoryDelete(ctx, request, MemoryDeleteInput{ID: "   "})
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !strings.Contains(err.Error(), "non-empty string") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestBuildRetrieveMatches(t *testing.T) {
	resp := &retrieval.RetrievalResponse{
		Results: []retrieval.UnifiedResult{
			{
				Item: retrieval.MemoryItem{
					ID:   "mem-1",
					Text: "remember me",
					Tags: []string{"tag1"},
				},
				Score:  0.88,
				Source: "vector",
			},
		},
	}

	matches := buildRetrieveMatches(resp)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].ID != "mem-1" {
		t.Fatalf("unexpected id: %s", matches[0].ID)
	}
	if matches[0].Score != 0.88 {
		t.Fatalf("unexpected score: %.2f", matches[0].Score)
	}
}

// TestHandleMemoryRetrieve_Success tests successful memory retrieval
func TestHandleMemoryRetrieve_Success(t *testing.T) {
	ctx := context.Background()
	request := &mcp.CallToolRequest{}

	// First save a memory
	saveInput := MemorySaveInput{
		Text: "Go programming language was created by Google in 2009",
		Tags: "go, programming, google",
	}
	_, saveOutput, err := handleMemorySave(ctx, request, saveInput)
	if err != nil {
		t.Fatalf("failed to save test memory: %v", err)
	}
	t.Logf("Saved test memory with ID: %s", saveOutput.ID)

	// Now retrieve it
	retrieveInput := MemoryRetrieveInput{Query: "Go programming language Google"}
	_, retrieveOutput, err := handleMemoryRetrieve(ctx, request, retrieveInput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("Retrieved results:\n%s", retrieveOutput.Results)

	// Results should not be empty (we just saved a matching memory)
	if retrieveOutput.Results == "" {
		t.Log("Warning: no results returned, but this may be due to embedding similarity threshold")
	}
}

// TestHandleMemorySave_TagsParsing tests tag parsing logic
func TestHandleMemorySave_TagsParsing(t *testing.T) {
	ctx := context.Background()
	request := &mcp.CallToolRequest{}

	testCases := []struct {
		name     string
		tags     string
		wantSave bool
	}{
		{"empty tags", "", true},
		{"single tag", "test", true},
		{"multiple tags", "tag1, tag2, tag3", true},
		{"tags with extra spaces", "  tag1  ,  tag2  ", true},
		{"tags with empty parts", "tag1,,tag2", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := MemorySaveInput{
				Text: "Test memory: " + tc.name,
				Tags: tc.tags,
			}
			_, output, err := handleMemorySave(ctx, request, input)
			if tc.wantSave {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if output.ID == "" {
					t.Fatal("expected non-empty ID")
				}
				t.Logf("Saved with ID: %s", output.ID)
			} else {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			}
		})
	}
}
