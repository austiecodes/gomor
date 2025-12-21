package commands

import (
	memorycmd "github.com/austiecodes/goa/internal/commands/memory"
)

func init() {
	rootCmd.AddCommand(memorycmd.MemoryCmd)
}

