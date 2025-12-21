package set

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/austiecodes/goa/internal/consts"
	"github.com/austiecodes/goa/internal/types"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateMainMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selected := m.List.SelectedItem().(MenuItem)
			switch selected.Title() {
			case "provider":
				m.List = createProviderList()
				m.Screen = ScreenProviderSelect
			case "chat-model":
				m.ModelType = ModelTypeChat
				m.List = createProviderList()
				m.Screen = ScreenModelProviderSelect
			case "title-model":
				m.ModelType = ModelTypeTitle
				m.List = createProviderList()
				m.Screen = ScreenModelProviderSelect
			case "think-model":
				m.ModelType = ModelTypeThink
				m.List = createProviderList()
				m.Screen = ScreenModelProviderSelect
			case "tool-model":
				m.ModelType = ModelTypeTool
				m.List = createProviderList()
				m.Screen = ScreenModelProviderSelect
			case "embedding-model":
				m.ModelType = ModelTypeEmbedding
				m.List = createProviderList()
				m.Screen = ScreenModelProviderSelect
			case "memory":
				m.TextInputs = createMemoryConfigInputs(m.Config)
				m.FocusedInput = 0
				m.Screen = ScreenMemoryConfig
				return *m, m.TextInputs[0].Focus()
			case "exit":
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
			if selected.Title() == consts.ProviderOpenAI {
				m.TextInputs = createProviderConfigInputs(m.Config)
				m.FocusedInput = 0
				m.Screen = ScreenProviderConfig
				return *m, m.TextInputs[0].Focus()
			}
			return *m, nil
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

			m.Config.Providers.OpenAI.APIKey = apiKey
			m.Config.Providers.OpenAI.BaseURL = baseURL

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
				Provider: consts.ProviderOpenAI,
				ModelID:  modelID,
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
			case ModelTypeEmbedding:
				m.Config.Model.EmbeddingModel = newModel
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

			ftsStrategy := strings.TrimSpace(m.TextInputs[3].Value())
			validStrategies := map[string]bool{"direct": true, "summary": true, "keywords": true, "auto": true}
			if !validStrategies[ftsStrategy] {
				m.Err = fmt.Errorf("fts_strategy must be one of: direct, summary, keywords, auto")
				return *m, nil
			}

			m.Config.Memory.MinSimilarity = minSim
			m.Config.Memory.MemoryTopK = memTopK
			m.Config.Memory.HistoryTopK = histTopK
			m.Config.Memory.FTSStrategy = ftsStrategy

			return *m, saveConfig(m.Config)
		}
	}

	// Update focused text input
	var cmd tea.Cmd
	m.TextInputs[m.FocusedInput], cmd = m.TextInputs[m.FocusedInput].Update(msg)
	return *m, cmd
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
		s.WriteString(TitleStyle.Render("Configure OpenAI Provider"))
		s.WriteString("\n\n")
		for i, input := range m.TextInputs {
			label := ""
			switch i {
			case 0:
				label = "API Key (required)"
			case 1:
				label = "Base URL (optional, default: OpenAI API)"
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
			"FTS Strategy (direct/summary/keywords/auto)",
		}
		for i, input := range m.TextInputs {
			s.WriteString(InputLabelStyle.Render(labels[i]))
			s.WriteString("\n")
			s.WriteString(input.View())
			s.WriteString("\n\n")
		}
		s.WriteString(HelpStyle.Render("Press Enter to save, Esc to cancel, Tab/Shift+Tab to navigate"))
	}

	if m.Err != nil {
		s.WriteString("\n\n")
		s.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.Err)))
	}

	return s.String()
}
