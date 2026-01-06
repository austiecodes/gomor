package commands

import (
	mcpcmd "github.com/austiecodes/gomor/internal/commands/mcp"
	memorycmd "github.com/austiecodes/gomor/internal/commands/memory"
	setcmd "github.com/austiecodes/gomor/internal/commands/set"
)

func init() {
	rootCmd.AddCommand(mcpcmd.McpCmd)
	rootCmd.AddCommand(memorycmd.MemoryCmd)
	rootCmd.AddCommand(setcmd.SetCmd)
}
