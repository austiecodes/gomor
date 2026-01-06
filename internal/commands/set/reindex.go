package set

import (
	"context"

	"github.com/austiecodes/gomor/internal/memory/retrieval"
	"github.com/austiecodes/gomor/internal/provider"
	"github.com/austiecodes/gomor/internal/types"
	"github.com/austiecodes/gomor/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
)

// ReindexResultMsg indicates the result of the reindexing process
type ReindexResultMsg struct {
	Err error
}

func reindexMemories(config *utils.Config, newModel types.Model) tea.Cmd {
	return func() tea.Msg {
		// 1. Initialize store
		s, err := retrieval.NewStore()
		if err != nil {
			return ReindexResultMsg{Err: err}
		}
		defer s.Close()

		// 2. Initialize embedding client
		// We need to use the provider from the new model
		// But we need the config for that provider.
		client, err := provider.NewEmbeddingClient(config, newModel.Provider)
		if err != nil {
			return ReindexResultMsg{Err: err}
		}

		// 3. Perform reindexing
		// We use a background context here, or could pass a context if available
		err = retrieval.ReindexMemories(context.Background(), s, client, newModel)
		return ReindexResultMsg{Err: err}
	}
}
