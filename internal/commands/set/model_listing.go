package set

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/austiecodes/gomor/internal/provider"
	"github.com/austiecodes/gomor/internal/utils"
)

func loadModelsForProvider(providerID string, cfg *utils.Config) tea.Cmd {
	return func() tea.Msg {
		c, err := provider.NewQueryClient(cfg, providerID)
		if err != nil {
			return ModelsLoadedMsg{Err: err}
		}

		models, err := c.ListModels(context.Background())
		return ModelsLoadedMsg{Models: models, Err: err}
	}
}
