package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"

	"github.com/austiecodes/gomor/internal/client"
	"github.com/austiecodes/gomor/internal/memory/retrieval"
	"github.com/austiecodes/gomor/internal/memory/store"
	"github.com/austiecodes/gomor/internal/provider"
	"github.com/austiecodes/gomor/internal/types"
	"github.com/austiecodes/gomor/internal/utils"
)

// McpCmd is the command to start the MCP server
var McpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the MCP server over stdio",
	Long:  `Start a Model Context Protocol (MCP) server that communicates over stdio. This allows gomor to be used as an MCP tool provider.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runMcpServer(); err != nil {
			fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
			os.Exit(1)
		}
	},
}

func runMcpServer() error {
	// Create the MCP server
	s := server.NewMCPServer(
		"gomor",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Register the goa_memory_save tool
	memorySaveTool := mcp.NewTool("goa_memory_save",
		mcp.WithDescription("Save a user preference or fact to memory. Use this to store declarative statements about user preferences, knowledge, or context."),
		mcp.WithString("text",
			mcp.Required(),
			mcp.Description("The preference or fact to save (e.g., 'User prefers TypeScript over JavaScript')"),
		),
		mcp.WithString("tags",
			mcp.Description("Comma-separated tags for categorization (optional)"),
		),
		mcp.WithNumber("confidence",
			mcp.Description("Confidence score from 0.0 to 1.0 (default: 1.0)"),
		),
	)
	s.AddTool(memorySaveTool, handleMemorySave)

	// Register the goa_memory_retrieve tool (unified hybrid search)
	memoryRetrieveTool := mcp.NewTool("goa_memory_retrieve",
		mcp.WithDescription("Retrieve relevant memories using hybrid search (vector similarity + full-text search). Combines LLM-transformed queries with multiple retrieval strategies for best results. Thresholds and limits are controlled by configuration."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The query to search for related memories"),
		),
	)
	s.AddTool(memoryRetrieveTool, handleMemoryRetrieve)

	// Start the stdio server
	return server.ServeStdio(s)
}

// handleMemorySave handles the goa_memory_save tool call
func handleMemorySave(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	if args == nil {
		return mcp.NewToolResultError("missing arguments"), nil
	}

	// Extract text (required)
	textArg, ok := args["text"]
	if !ok {
		return mcp.NewToolResultError("missing required parameter: text"), nil
	}
	text, ok := textArg.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return mcp.NewToolResultError("parameter 'text' must be a non-empty string"), nil
	}

	// Extract tags (optional)
	var tags []string
	if tagsArg, ok := args["tags"]; ok {
		if tagsStr, ok := tagsArg.(string); ok && tagsStr != "" {
			for _, t := range strings.Split(tagsStr, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
		}
	}

	// Extract confidence (optional, default 1.0)
	confidence := 1.0
	if confArg, ok := args["confidence"]; ok {
		if confNum, ok := confArg.(float64); ok && confNum >= 0 && confNum <= 1 {
			confidence = confNum
		}
	}

	// Load config for embedding
	config, err := utils.LoadConfig()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load config: %v", err)), nil
	}

	if config.Model.EmbeddingModel == nil {
		return mcp.NewToolResultError("embedding model not configured. Run 'gomor set' to configure"), nil
	}

	// Create embedding client
	embeddingModel := *config.Model.EmbeddingModel
	embClient, err := provider.NewEmbeddingClient(config, embeddingModel.Provider)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create embedding client: %v", err)), nil
	}

	// Generate embedding
	embedding, err := embClient.Embed(ctx, embeddingModel, text)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to generate embedding: %v", err)), nil
	}

	// Normalize embedding for cosine similarity
	normalizedEmbedding := store.NormalizeVector(embedding)

	// Open memory store
	memStore, err := store.NewStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to open memory store: %v", err)), nil
	}
	defer memStore.Close()

	// Save memory
	item := &store.MemoryItem{
		Text:       text,
		Tags:       tags,
		Source:     store.SourceExplicit,
		Confidence: confidence,
		Provider:   embeddingModel.Provider,
		ModelID:    embeddingModel.ModelID,
		Dim:        len(normalizedEmbedding),
		Embedding:  normalizedEmbedding,
	}

	if err := memStore.SaveMemory(item); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to save memory: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Memory saved successfully (id: %s)", item.ID)), nil
}

// handleMemoryRetrieve handles the goa_memory_retrieve tool call (unified hybrid search)
func handleMemoryRetrieve(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	if args == nil {
		return mcp.NewToolResultError("missing arguments"), nil
	}

	// Extract query (required)
	queryArg, ok := args["query"]
	if !ok {
		return mcp.NewToolResultError("missing required parameter: query"), nil
	}
	query, ok := queryArg.(string)
	if !ok || strings.TrimSpace(query) == "" {
		return mcp.NewToolResultError("parameter 'query' must be a non-empty string"), nil
	}

	// Load config
	config, err := utils.LoadConfig()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load config: %v", err)), nil
	}

	if config.Model.EmbeddingModel == nil {
		return mcp.NewToolResultError("embedding model not configured. Run 'gomor set' to configure"), nil
	}

	// Open memory store
	memStore, err := store.NewStore()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to open memory store: %v", err)), nil
	}
	defer memStore.Close()

	// Create embedding client
	embeddingModel := *config.Model.EmbeddingModel
	embClient, err := provider.NewEmbeddingClient(config, embeddingModel.Provider)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create embedding client: %v", err)), nil
	}

	// Create query client for LLM transformations (optional, may be nil)
	var queryClient client.QueryClient
	toolModel := types.Model{}
	if config.Model.ToolModel != nil {
		toolModel = *config.Model.ToolModel
		queryClient, _ = provider.NewQueryClient(config, toolModel.Provider)
	}

	// Create retriever
	ret := retrieval.NewRetriever(
		memStore,
		embClient,
		queryClient,
		embeddingModel,
		toolModel,
		config.Memory,
	)

	// Perform retrieval
	response, err := ret.Retrieve(ctx, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("retrieval failed: %v", err)), nil
	}

	// Format results
	result := retrieval.FormatAsText(response)
	return mcp.NewToolResultText(result), nil
}
