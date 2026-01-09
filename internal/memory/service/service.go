package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/austiecodes/gomor/internal/client"
	"github.com/austiecodes/gomor/internal/memory/memtypes"
	"github.com/austiecodes/gomor/internal/memory/memutils"
	"github.com/austiecodes/gomor/internal/memory/retrieval"
	"github.com/austiecodes/gomor/internal/memory/store"
	"github.com/austiecodes/gomor/internal/provider"
	"github.com/austiecodes/gomor/internal/types"
	"github.com/austiecodes/gomor/internal/utils"
)

type SaveInput struct {
	Text   string
	Tags   []string
	Source memtypes.MemorySource
}

type SaveResult struct {
	Item memtypes.MemoryItem
}

type RetrieveInput struct {
	Query string
}

type RetrieveResult struct {
	Response *retrieval.RetrievalResponse
	Text     string
}

type DeleteInput struct {
	ID string
}

type DeleteResult struct {
	ID string
}

func Save(ctx context.Context, input SaveInput) (*SaveResult, error) {
	text := strings.TrimSpace(input.Text)
	if text == "" {
		return nil, fmt.Errorf("parameter 'text' must be a non-empty string")
	}

	config, err := utils.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if config.Model.EmbeddingModel == nil {
		return nil, fmt.Errorf("embedding model not configured. Run 'gomor set' to configure")
	}

	embeddingModel := *config.Model.EmbeddingModel
	embClient, err := provider.NewEmbeddingClient(config, embeddingModel.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding client: %w", err)
	}

	embedding, err := embClient.Embed(ctx, embeddingModel, text)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	memStore, err := store.NewStore()
	if err != nil {
		return nil, fmt.Errorf("failed to open memory store: %w", err)
	}
	defer memStore.Close()

	source := input.Source
	if source == "" {
		source = memtypes.SourceExplicit
	}

	item := memtypes.MemoryItem{
		Text:      text,
		Tags:      input.Tags,
		Source:    source,
		Provider:  embeddingModel.Provider,
		ModelID:   embeddingModel.ModelID,
		Dim:       len(embedding),
		Embedding: memutils.NormalizeVector(embedding),
	}

	if err := memStore.SaveMemory(&item); err != nil {
		return nil, fmt.Errorf("failed to save memory: %w", err)
	}

	return &SaveResult{Item: item}, nil
}

func Retrieve(ctx context.Context, input RetrieveInput) (*RetrieveResult, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return nil, fmt.Errorf("parameter 'query' must be a non-empty string")
	}

	config, err := utils.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if config.Model.EmbeddingModel == nil {
		return nil, fmt.Errorf("embedding model not configured. Run 'gomor set' to configure")
	}

	memStore, err := store.NewStore()
	if err != nil {
		return nil, fmt.Errorf("failed to open memory store: %w", err)
	}
	defer memStore.Close()

	embeddingModel := *config.Model.EmbeddingModel
	embClient, err := provider.NewEmbeddingClient(config, embeddingModel.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding client: %w", err)
	}

	queryClient, toolModel := buildQueryClient(config)

	ret := retrieval.NewRetriever(
		memStore,
		embClient,
		queryClient,
		embeddingModel,
		toolModel,
		config.Memory,
	)

	response, err := ret.Retrieve(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	return &RetrieveResult{
		Response: response,
		Text:     retrieval.FormatAsText(response),
	}, nil
}

func Delete(ctx context.Context, input DeleteInput) (*DeleteResult, error) {
	_ = ctx

	id := strings.TrimSpace(input.ID)
	if id == "" {
		return nil, fmt.Errorf("parameter 'id' must be a non-empty string")
	}

	memStore, err := store.NewStore()
	if err != nil {
		return nil, fmt.Errorf("failed to open memory store: %w", err)
	}
	defer memStore.Close()

	if err := memStore.DeleteMemory(id); err != nil {
		return nil, fmt.Errorf("failed to delete memory: %w", err)
	}

	return &DeleteResult{ID: id}, nil
}

func buildQueryClient(config *utils.Config) (client.QueryClient, types.Model) {
	if config.Model.ToolModel == nil {
		return nil, types.Model{}
	}

	toolModel := *config.Model.ToolModel
	queryClient, err := provider.NewQueryClient(config, toolModel.Provider)
	if err != nil {
		return nil, toolModel
	}

	return queryClient, toolModel
}
