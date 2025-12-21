package set

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/austiecodes/goa/internal/consts"
	"github.com/austiecodes/goa/internal/utils"
)

func createMainMenu() list.Model {
	items := []list.Item{
		MenuItem{title: "provider", desc: "Configure provider settings (API key, base URL)"},
		MenuItem{title: "chat-model", desc: "Set default model for chat"},
		MenuItem{title: "title-model", desc: "Set model for generating conversation titles"},
		MenuItem{title: "think-model", desc: "Set model for thinking"},
		MenuItem{title: "tool-model", desc: "Set model for tool/auxiliary prompts"},
		MenuItem{title: "embedding-model", desc: "Set model for embeddings"},
		MenuItem{title: "memory", desc: "Configure memory retrieval settings"},
		MenuItem{title: "exit", desc: "Exit settings"},
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 60, 30)
	l.Title = "Goa Settings"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(true)
	return l
}

func createProviderList() list.Model {
	items := []list.Item{
		MenuItem{title: consts.ProviderOpenAI, desc: "OpenAI API (GPT models)"},
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 60, 10)
	l.Title = "Select Provider"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(true)
	return l
}

func createProviderConfigInputs(config *utils.Config) []textinput.Model {
	inputs := make([]textinput.Model, 2)

	// API Key input
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "sk-..."
	inputs[0].EchoMode = textinput.EchoPassword
	inputs[0].EchoCharacter = '*'
	inputs[0].CharLimit = 256
	inputs[0].Width = 50
	if config.Providers.OpenAI.APIKey != "" {
		inputs[0].SetValue(config.Providers.OpenAI.APIKey)
	}

	// Base URL input
	inputs[1] = textinput.New()
	inputs[1].Placeholder = consts.DefaultBaseURL
	inputs[1].CharLimit = 256
	inputs[1].Width = 50
	if config.Providers.OpenAI.BaseURL != "" {
		inputs[1].SetValue(config.Providers.OpenAI.BaseURL)
	}

	return inputs
}

func createModelList(models []string, mt ModelType) list.Model {
	items := make([]list.Item, len(models))
	for i, modelID := range models {
		items[i] = MenuItem{title: modelID, desc: ""}
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 60, 30)

	switch mt {
	case ModelTypeChat:
		l.Title = "Select Chat Model"
	case ModelTypeTitle:
		l.Title = "Select Title Model"
	case ModelTypeThink:
		l.Title = "Select Think Model"
	case ModelTypeTool:
		l.Title = "Select Tool Model"
	case ModelTypeEmbedding:
		l.Title = "Select Embedding Model"
	}

	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)
	return l
}

func createMemoryConfigInputs(config *utils.Config) []textinput.Model {
	inputs := make([]textinput.Model, 4)

	// Min Similarity input
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "0.80"
	inputs[0].CharLimit = 10
	inputs[0].Width = 20
	inputs[0].SetValue(formatFloat(config.Memory.MinSimilarity))

	// Memory TopK input
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "10"
	inputs[1].CharLimit = 5
	inputs[1].Width = 20
	inputs[1].SetValue(formatInt(config.Memory.MemoryTopK))

	// History TopK input
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "10"
	inputs[2].CharLimit = 5
	inputs[2].Width = 20
	inputs[2].SetValue(formatInt(config.Memory.HistoryTopK))

	// FTS Strategy input
	inputs[3] = textinput.New()
	inputs[3].Placeholder = "direct"
	inputs[3].CharLimit = 20
	inputs[3].Width = 20
	if config.Memory.FTSStrategy != "" {
		inputs[3].SetValue(config.Memory.FTSStrategy)
	} else {
		inputs[3].SetValue("direct")
	}

	return inputs
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

func formatInt(i int) string {
	return fmt.Sprintf("%d", i)
}

func saveConfig(config *utils.Config) tea.Cmd {
	return func() tea.Msg {
		err := utils.SaveConfig(config)
		return ConfigSavedMsg{Err: err}
	}
}
