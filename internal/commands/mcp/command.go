package mcp

import (
	"context"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
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
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "gomor",
			Version: "0.7.0",
		},
		nil,
	)

	// Register the memory_save tool
	memorySaveTool := &mcp.Tool{
		Name:        "memory_save",
		Description: "Save a user preference or fact to memory. Use this to store declarative statements about user preferences, knowledge, or context.",
	}
	mcp.AddTool(server, memorySaveTool, handleMemorySave)

	// Register the memory_retrieve tool
	memoryRetrieveTool := &mcp.Tool{
		Name:        "memory_retrieve",
		Description: "Retrieve relevant memories based on a query. Use this to recall user preferences, facts, or context that was previously saved.",
	}
	mcp.AddTool(server, memoryRetrieveTool, handleMemoryRetrieve)

	// Register the memory_delete tool
	memoryDeleteTool := &mcp.Tool{
		Name:        "memory_delete",
		Description: "Delete an incorrect or obsolete memory by ID.",
	}
	mcp.AddTool(server, memoryDeleteTool, handleMemoryDelete)

	// Start the stdio server
	ctx := context.Background()
	return server.Run(ctx, &mcp.StdioTransport{})
}
