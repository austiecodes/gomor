package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/austiecodes/gomor/internal/memory/memtypes"
	"github.com/austiecodes/gomor/internal/memory/retrieval"
	memoryservice "github.com/austiecodes/gomor/internal/memory/service"
)

func TestMemoryCommandNoFlagsRunsInteractive(t *testing.T) {
	oldRunInteractive := runInteractiveMemory
	defer func() { runInteractiveMemory = oldRunInteractive }()

	called := false
	runInteractiveMemory = func() error {
		called = true
		return nil
	}

	cmd := newMemoryCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !called {
		t.Fatal("expected interactive memory UI to run")
	}
}

func TestMemoryCommandRejectsMutuallyExclusiveFlags(t *testing.T) {
	cmd := newMemoryCommand()
	cmd.SetArgs([]string{"--save", "remember", "--query", "remember"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected mutually exclusive flag error")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMemoryCommandRejectsTagsWithoutSave(t *testing.T) {
	cmd := newMemoryCommand()
	cmd.SetArgs([]string{"--query", "remember", "--tags", "tag1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected tags validation error")
	}
	if !strings.Contains(err.Error(), "--tags can only be used with --save") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMemoryCommandSaveJSONOutput(t *testing.T) {
	oldSaveMemory := saveMemoryFn
	defer func() { saveMemoryFn = oldSaveMemory }()

	saveMemoryFn = func(ctx context.Context, input memoryservice.SaveInput) (*memoryservice.SaveResult, error) {
		return &memoryservice.SaveResult{
			Item: memtypes.MemoryItem{
				ID:   "mem-1",
				Text: input.Text,
				Tags: input.Tags,
			},
		}, nil
	}

	cmd := newMemoryCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--save", "remember this", "--tags", "tag1, tag2", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var payload memorySaveOutput
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if payload.ID != "mem-1" {
		t.Fatalf("unexpected id: %s", payload.ID)
	}
}

func TestMemoryCommandQueryJSONOutput(t *testing.T) {
	oldQueryMemory := queryMemoryFn
	defer func() { queryMemoryFn = oldQueryMemory }()

	queryMemoryFn = func(ctx context.Context, input memoryservice.RetrieveInput) (*memoryservice.RetrieveResult, error) {
		return &memoryservice.RetrieveResult{
			Text: "Found 1 memories:\n\n1. [0.88] remember this",
			Response: &retrieval.RetrievalResponse{
				Results: []retrieval.UnifiedResult{
					{
						Item: retrieval.MemoryItem{
							ID:   "mem-1",
							Text: "remember this",
							Tags: []string{"tag1"},
						},
						Score:  0.88,
						Source: "vector",
					},
				},
			},
		}, nil
	}

	cmd := newMemoryCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--query", "remember", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var payload memoryQueryOutput
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if len(payload.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(payload.Matches))
	}
	if payload.Matches[0].ID != "mem-1" {
		t.Fatalf("unexpected id: %s", payload.Matches[0].ID)
	}
}

func TestMemoryCommandDeleteJSONOutput(t *testing.T) {
	oldDeleteMemory := deleteMemoryFn
	defer func() { deleteMemoryFn = oldDeleteMemory }()

	deleteMemoryFn = func(ctx context.Context, input memoryservice.DeleteInput) (*memoryservice.DeleteResult, error) {
		return &memoryservice.DeleteResult{
			ID:      input.ID,
			Deleted: true,
		}, nil
	}

	cmd := newMemoryCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--delete", "mem-1", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var payload memoryDeleteOutput
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if !payload.Deleted {
		t.Fatal("expected deleted=true")
	}
	if payload.ID != "mem-1" {
		t.Fatalf("unexpected id: %s", payload.ID)
	}
}
