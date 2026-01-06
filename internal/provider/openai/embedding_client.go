package openai

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"

	"github.com/austiecodes/gomor/internal/client"
	"github.com/austiecodes/gomor/internal/types"
)

// EmbeddingClient wraps OpenAI client for embedding operations.
type EmbeddingClient struct {
	c *Client
}

// Compile-time check that EmbeddingClient implements client.EmbeddingClient.
var _ client.EmbeddingClient = (*EmbeddingClient)(nil)

// NewEmbeddingClient creates a new OpenAI embedding client.
func NewEmbeddingClient(apiKey, baseURL string) *EmbeddingClient {
	return &EmbeddingClient{c: NewClient(apiKey, baseURL)}
}

// Embed returns the embedding vector for the given text.
func (e *EmbeddingClient) Embed(ctx context.Context, model types.Model, text string) ([]float32, error) {
	resp, err := e.c.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(model.ModelID),
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: param.NewOpt(text),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("openai embedding failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("openai returned empty embedding data")
	}

	// Convert float64 to float32 for more compact storage
	embedding := make([]float32, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// EmbedBatch returns embedding vectors for multiple texts.
func (e *EmbeddingClient) EmbedBatch(ctx context.Context, model types.Model, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	resp, err := e.c.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(model.ModelID),
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("openai batch embedding failed: %w", err)
	}

	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("openai returned %d embeddings, expected %d", len(resp.Data), len(texts))
	}

	// Convert float64 to float32 for more compact storage
	result := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embedding := make([]float32, len(data.Embedding))
		for j, v := range data.Embedding {
			embedding[j] = float32(v)
		}
		result[i] = embedding
	}

	return result, nil
}

// Dimensions returns the embedding dimension for the given model.
// This is a static lookup based on known OpenAI embedding models.
func (e *EmbeddingClient) Dimensions(model types.Model) int {
	// Known OpenAI embedding model dimensions
	switch model.ModelID {
	case string(openai.EmbeddingModelTextEmbedding3Small):
		return 1536
	case string(openai.EmbeddingModelTextEmbedding3Large):
		return 3072
	case string(openai.EmbeddingModelTextEmbeddingAda002):
		return 1536
	default:
		// Default to text-embedding-3-small dimensions
		return 1536
	}
}
