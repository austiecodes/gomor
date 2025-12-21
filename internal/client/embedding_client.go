package client

import (
	"context"

	"github.com/austiecodes/goa/internal/types"
)

// EmbeddingClient is the interface for generating embeddings from text.
type EmbeddingClient interface {
	// Embed returns the embedding vector for the given text using the specified model.
	Embed(ctx context.Context, model types.Model, text string) ([]float32, error)

	// EmbedBatch returns embedding vectors for multiple texts using the specified model.
	EmbedBatch(ctx context.Context, model types.Model, texts []string) ([][]float32, error)

	// Dimensions returns the embedding dimension for the given model.
	Dimensions(model types.Model) int
}

