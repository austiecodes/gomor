package google

import (
	"context"
	"fmt"

	"github.com/austiecodes/gomor/internal/client"
	"github.com/austiecodes/gomor/internal/types"
	"google.golang.org/genai"
)

// EmbeddingClient wraps Google Gemini client for embedding operations.
type EmbeddingClient struct {
	c *Client
}

// Compile-time check that EmbeddingClient implements client.EmbeddingClient.
var _ client.EmbeddingClient = (*EmbeddingClient)(nil)

// NewEmbeddingClient creates a new Google embedding client.
func NewEmbeddingClient(apiKey, baseURL string) *EmbeddingClient {
	return &EmbeddingClient{c: NewClient(apiKey, baseURL)}
}

// Embed returns the embedding vector for the given text.
func (e *EmbeddingClient) Embed(ctx context.Context, model types.Model, text string) ([]float32, error) {
	if e.c == nil || e.c.client == nil {
		return nil, fmt.Errorf("google client not initialized")
	}

	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: text},
			},
		},
	}

	resp, err := e.c.client.Models.EmbedContent(ctx, model.ModelID, contents, nil)
	if err != nil {
		return nil, fmt.Errorf("google embedding failed: %w", err)
	}

	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("google returned empty embedding data")
	}

	return resp.Embeddings[0].Values, nil
}

// EmbedBatch returns embedding vectors for multiple texts.
func (e *EmbeddingClient) EmbedBatch(ctx context.Context, model types.Model, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	if e.c == nil || e.c.client == nil {
		return nil, fmt.Errorf("google client not initialized")
	}

	contents := make([]*genai.Content, len(texts))
	for i, text := range texts {
		contents[i] = &genai.Content{
			Parts: []*genai.Part{
				{Text: text},
			},
		}
	}

	resp, err := e.c.client.Models.EmbedContent(ctx, model.ModelID, contents, nil)
	if err != nil {
		return nil, fmt.Errorf("google batch embedding failed: %w", err)
	}

	if len(resp.Embeddings) != len(texts) {
		return nil, fmt.Errorf("google returned %d embeddings, expected %d", len(resp.Embeddings), len(texts))
	}

	result := make([][]float32, len(resp.Embeddings))
	for i, emb := range resp.Embeddings {
		result[i] = emb.Values
	}

	return result, nil
}

// Dimensions returns the embedding dimension for the given model.
func (e *EmbeddingClient) Dimensions(model types.Model) int {
	// Known Google embedding model dimensions
	switch model.ModelID {
	case "text-embedding-004":
		return 768
	case "text-multilingual-embedding-002":
		return 768
	default:
		// Default to 768 as it's common for Gemini embeddings
		return 768
	}
}
