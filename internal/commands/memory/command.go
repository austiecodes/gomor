package memory

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var MemoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Manage memories interactively",
	Long:  `Open an interactive TUI to view, add, edit, and delete stored memories.`,
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(initialModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running memory manager: %v\n", err)
		}
	},
}

