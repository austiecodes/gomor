package commands

import (
	mcpcmd "github.com/austiecodes/goa/internal/commands/mcp"
)

func init() {
	rootCmd.AddCommand(mcpcmd.McpCmd)
}

