package client

import (
	"context"

	"github.com/austiecodes/gomor/internal/types"
)

type Message interface {
	GetRole() string
	GetContent() any
}

type ChatRequest interface {
	GetModel() types.Model
}

type ChatResponse interface {
	GetContent() any
}

// QueryClient is a high-level client interface for simple "prompt in, stream out" workflows.
// It is designed to keep commands extensible without exposing provider-specific request types.
type QueryClient interface {
	// ChatStream streams the response for a single user query under the given model.
	ChatStream(ctx context.Context, model types.Model, query string) (StreamResponse, error)
	// ChatStreamWithContext streams the response with optional system context prefix.
	ChatStreamWithContext(ctx context.Context, model types.Model, systemContext, query string) (StreamResponse, error)
	// ListModels lists models available to this client/provider.
	ListModels(ctx context.Context) ([]string, error)
}

// StreamResponse is the interface for streaming chat responses
type StreamResponse interface {
	// Next advances to the next chunk, returns true if there is more data
	Next() bool
	// GetChunk returns the content of the current chunk
	GetChunk() string
	// Err returns any error encountered during iteration
	Err() error
	// Close closes the stream and releases resources
	Close() error
}

// Client is the interface that all LLM providers must implement
type Client interface {
	Chat(ctx context.Context, request ChatRequest) (ChatResponse, error)
	ChatStream(ctx context.Context, request ChatRequest) (StreamResponse, error)
}
