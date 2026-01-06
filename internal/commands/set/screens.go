package set

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/austiecodes/gomor/internal/consts"
	"github.com/austiecodes/gomor/internal/types"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateMainMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selected := m.List.SelectedItem().(MenuItem)
			switch selected.Title() {
			case MenuItemProvider:
				m.List = createProviderList()
				m.Screen = ScreenProviderSelect
			case MenuItemChatModel:
				m.ModelType = ModelTypeChat
				m.List = createProviderList()
				m.Screen = ScreenModelProviderSelect
			case MenuItemTitleModel:
				m.ModelType = ModelTypeTitle
				m.List = createProviderList()
				m.Screen = ScreenModelProviderSelect
			case MenuItemThinkModel:
				m.ModelType = ModelTypeThink
				m.List = createProviderList()
				m.Screen = ScreenModelProviderSelect
			case MenuItemToolModel:
				m.ModelType = ModelTypeTool
				m.List = createProviderList()
				m.Screen = ScreenModelProviderSelect
			case MenuItemEmbeddingModel:
				m.ModelType = ModelTypeEmbedding
				m.List = createProviderList()
				m.Screen = ScreenModelProviderSelect
			case MenuItemMemory:
				m.TextInputs = createMemoryConfigInputs(m.Config)
				m.FocusedInput = 0
				m.Screen = ScreenMemoryConfig
				return *m, m.TextInputs[0].Focus()
			case MenuItemExit:
				m.Quitting = true
				return *m, tea.Quit
			}
			return *m, nil
		}
	}

	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return *m, cmd
}

func (m *Model) updateProviderSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selected := m.List.SelectedItem().(MenuItem)
			m.TextInputs = createProviderConfigInputs(m.Config, selected.Title())
			m.FocusedInput = 0
			m.Screen = ScreenProviderConfig
			return *m, m.TextInputs[0].Focus()
		}
	}

	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return *m, cmd
}

func (m *Model) updateProviderConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			m.TextInputs[m.FocusedInput].Blur()
			m.FocusedInput = (m.FocusedInput + 1) % len(m.TextInputs)
			return *m, m.TextInputs[m.FocusedInput].Focus()

		case "shift+tab", "up":
			m.TextInputs[m.FocusedInput].Blur()
			m.FocusedInput = (m.FocusedInput - 1 + len(m.TextInputs)) % len(m.TextInputs)
			return *m, m.TextInputs[m.FocusedInput].Focus()

		case "enter":
			// Save config
			apiKey := m.TextInputs[0].Value()
			baseURL := m.TextInputs[1].Value()

			if apiKey == "" {
				m.Err = fmt.Errorf("API key is required")
				return *m, nil
			}

			provider := m.List.SelectedItem().(MenuItem).Title()
			switch provider {
			case consts.ProviderOpenAI:
				m.Config.Providers.OpenAI.APIKey = apiKey
				m.Config.Providers.OpenAI.BaseURL = baseURL
			case consts.ProviderGoogle:
				m.Config.Providers.Google.APIKey = apiKey
				m.Config.Providers.Google.BaseURL = baseURL
			case consts.ProviderAnthropic:
				m.Config.Providers.Anthropic.APIKey = apiKey
				m.Config.Providers.Anthropic.BaseURL = baseURL
			}

			return *m, saveConfig(m.Config)
		}
	}

	// Update focused text input
	var cmd tea.Cmd
	m.TextInputs[m.FocusedInput], cmd = m.TextInputs[m.FocusedInput].Update(msg)
	return *m, cmd
}

func (m *Model) updateModelProviderSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selected := m.List.SelectedItem().(MenuItem)
			providerID := selected.Title()
			m.SelectedProvider = providerID
			return *m, loadModelsForProvider(providerID, m.Config)
		}
	}

	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return *m, cmd
}

func (m *Model) updateModelSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selected := m.List.SelectedItem().(MenuItem)
			modelID := selected.Title()

			newModel := &types.Model{
				Provider: m.SelectedProvider,
				ModelID:  modelID,
			}

			if m.ModelType == ModelTypeEmbedding {
				// Check if model actually changed
				oldModel := m.Config.Model.EmbeddingModel
				if oldModel != nil && oldModel.ModelID == newModel.ModelID && oldModel.Provider == newModel.Provider {
					// No change, just go back
					m.Screen = ScreenMainMenu
					m.List = createMainMenu()
					return *m, nil
				}

				// Model changed, ask for confirmation
				m.PendingModel = newModel
				m.Screen = ScreenConfirmReindex
				return *m, nil
			}

			switch m.ModelType {
			case ModelTypeChat:
				m.Config.Model.ChatModel = newModel
			case ModelTypeTitle:
				m.Config.Model.TitleModel = newModel
			case ModelTypeThink:
				m.Config.Model.ThinkModel = newModel
			case ModelTypeTool:
				m.Config.Model.ToolModel = newModel
			}

			return *m, saveConfig(m.Config)
		}
	}

	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return *m, cmd
}

func (m *Model) updateMemoryConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			m.TextInputs[m.FocusedInput].Blur()
			m.FocusedInput = (m.FocusedInput + 1) % len(m.TextInputs)
			return *m, m.TextInputs[m.FocusedInput].Focus()

		case "shift+tab", "up":
			m.TextInputs[m.FocusedInput].Blur()
			m.FocusedInput = (m.FocusedInput - 1 + len(m.TextInputs)) % len(m.TextInputs)
			return *m, m.TextInputs[m.FocusedInput].Focus()

		case "enter":
			// Parse and save memory config
			minSim, err := strconv.ParseFloat(m.TextInputs[0].Value(), 64)
			if err != nil || minSim < 0 || minSim > 1 {
				m.Err = fmt.Errorf("min_similarity must be a number between 0 and 1")
				return *m, nil
			}

			memTopK, err := strconv.Atoi(m.TextInputs[1].Value())
			if err != nil || memTopK < 1 {
				m.Err = fmt.Errorf("memory_top_k must be a positive integer")
				return *m, nil
			}

			histTopK, err := strconv.Atoi(m.TextInputs[2].Value())
			if err != nil || histTopK < 1 {
				m.Err = fmt.Errorf("history_top_k must be a positive integer")
				return *m, nil
			}

			m.Config.Memory.MinSimilarity = minSim
			m.Config.Memory.MemoryTopK = memTopK
			m.Config.Memory.HistoryTopK = histTopK

			return *m, saveConfig(m.Config)
		}
	}

	// Update focused text input
	var cmd tea.Cmd
	m.TextInputs[m.FocusedInput], cmd = m.TextInputs[m.FocusedInput].Update(msg)
	return *m, cmd
}

func (m *Model) updateConfirmReindex(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.Reindexing {
		// Handle reindexing completion
		switch msg := msg.(type) {
		case ReindexResultMsg:
			m.Reindexing = false
			if msg.Err != nil {
				m.Err = msg.Err
				// Go back to main menu with error
				m.Screen = ScreenMainMenu
				m.List = createMainMenu()
			} else {
				// Success, save everything; PendingModel is now the current model
				m.Config.Model.EmbeddingModel = m.PendingModel
				m.PendingModel = nil
				return *m, saveConfig(m.Config)
			}
			return *m, nil
		}
		// Ignore keys while processing
		return *m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.Reindexing = true
			return *m, reindexMemories(m.Config, *m.PendingModel)
		case "n", "N", "esc":
			m.PendingModel = nil
			m.Screen = ScreenMainMenu
			m.List = createMainMenu()
			return *m, nil
		}
	}
	return *m, nil
}

func (m *Model) renderView() string {
	if m.Quitting {
		return "Goodbye!\n"
	}

	var s strings.Builder

	switch m.Screen {
	case ScreenMainMenu:
		s.WriteString(m.List.View())

	case ScreenProviderSelect:
		s.WriteString(TitleStyle.Render("Select Provider"))
		s.WriteString("\n\n")
		s.WriteString(m.List.View())

	case ScreenProviderConfig:
		provider := m.List.SelectedItem().(MenuItem).Title()
		s.WriteString(TitleStyle.Render(fmt.Sprintf("Configure %s Provider", provider)))
		s.WriteString("\n\n")
		for i, input := range m.TextInputs {
			label := ""
			switch i {
			case 0:
				label = "API Key (required)"
			case 1:
				label = "Base URL (optional, default: Provider Default)"
			}
			s.WriteString(InputLabelStyle.Render(label))
			s.WriteString("\n")
			s.WriteString(input.View())
			s.WriteString("\n\n")
		}
		s.WriteString(HelpStyle.Render("Press Enter to save, Esc to cancel, Tab/Shift+Tab to navigate"))

	case ScreenModelProviderSelect:
		modelName := ""
		switch m.ModelType {
		case ModelTypeChat:
			modelName = "Chat Model"
		case ModelTypeTitle:
			modelName = "Title Model"
		case ModelTypeThink:
			modelName = "Think Model"
		case ModelTypeTool:
			modelName = "Tool Model"
		case ModelTypeEmbedding:
			modelName = "Embedding Model"
		}
		s.WriteString(TitleStyle.Render(fmt.Sprintf("Select Provider for %s", modelName)))
		s.WriteString("\n\n")
		s.WriteString(m.List.View())

	case ScreenModelSelect:
		modelName := ""
		switch m.ModelType {
		case ModelTypeChat:
			modelName = "Chat Model"
		case ModelTypeTitle:
			modelName = "Title Model"
		case ModelTypeThink:
			modelName = "Think Model"
		case ModelTypeTool:
			modelName = "Tool Model"
		case ModelTypeEmbedding:
			modelName = "Embedding Model"
		}
		s.WriteString(TitleStyle.Render(fmt.Sprintf("Select %s", modelName)))
		s.WriteString("\n\n")
		s.WriteString(m.List.View())

	case ScreenMemoryConfig:
		s.WriteString(TitleStyle.Render("Memory Retrieval Settings"))
		s.WriteString("\n\n")
		labels := []string{
			"Min Similarity (0.0-1.0, default: 0.80)",
			"Memory Top K (default: 10)",
			"History Top K (default: 10)",
		}
		for i, input := range m.TextInputs {
			s.WriteString(InputLabelStyle.Render(labels[i]))
			s.WriteString("\n")
			s.WriteString(input.View())
			s.WriteString("\n\n")
		}
		s.WriteString(HelpStyle.Render("Press Enter to save, Esc to cancel, Tab/Shift+Tab to navigate"))

	case ScreenConfirmReindex:
		if m.Reindexing {
			s.WriteString(TitleStyle.Render("Reindexing Memories..."))
			s.WriteString("\n\n")
			s.WriteString("Please wait while we update your memory embeddings.")
			s.WriteString("\nThis may take a while depending on the number of memories.")
		} else {
			s.WriteString(TitleStyle.Render("Confirm Embedding Model Change"))
			s.WriteString("\n\n")
			s.WriteString("Changing the embedding model requires reindexing all existing memories.\n")
			s.WriteString("This process will re-calculate embeddings for all items using the new model.\n\n")
			s.WriteString("Do you want to proceed?\n\n")
			s.WriteString(HelpStyle.Render("Press 'y' to confirm and reindex, 'n' to cancel"))
		}
	}

	if m.Err != nil {
		s.WriteString("\n\n")
		s.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.Err)))
	}

	return s.String()
}
