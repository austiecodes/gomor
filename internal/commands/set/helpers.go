package set

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/austiecodes/gomor/internal/consts"
	"github.com/austiecodes/gomor/internal/utils"
)

func createMainMenu() list.Model {
	items := []list.Item{
		MenuItem{title: MenuItemProvider, desc: "Configure provider settings (API key, base URL)"},
		MenuItem{title: MenuItemChatModel, desc: "Set default model for chat"},
		MenuItem{title: MenuItemTitleModel, desc: "Set model for generating conversation titles"},
		MenuItem{title: MenuItemThinkModel, desc: "Set model for thinking"},
		MenuItem{title: MenuItemToolModel, desc: "Set model for tool/auxiliary prompts"},
		MenuItem{title: MenuItemEmbeddingModel, desc: "Set model for embeddings"},
		MenuItem{title: MenuItemMemory, desc: "Configure memory retrieval settings"},
		MenuItem{title: MenuItemExit, desc: "Exit settings"},
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 60, 30)
	l.Title = "gomor Settings"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(true)
	return l
}

func createProviderList() list.Model {
	items := []list.Item{
		MenuItem{title: consts.ProviderOpenAI, desc: "OpenAI API (GPT models)"},
		MenuItem{title: consts.ProviderGoogle, desc: "Google Gemini API (GEMINI models)"},
		MenuItem{title: consts.ProviderAnthropic, desc: "Anthropic API (Claude models)"},
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 60, 15)
	l.Title = "Select Provider"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(true)
	return l
}

func createProviderConfigInputs(config *utils.Config, provider string) []textinput.Model {
	inputs := make([]textinput.Model, 2)

	// API Key input
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "sk-..."
	inputs[0].EchoMode = textinput.EchoPassword
	inputs[0].EchoCharacter = '*'
	inputs[0].CharLimit = 256
	inputs[0].Width = 50

	// Base URL input
	inputs[1] = textinput.New()
	inputs[1].CharLimit = 256
	inputs[1].Width = 50

	var apiKey, baseURL string
	switch provider {
	case consts.ProviderOpenAI:
		inputs[1].Placeholder = consts.DefaultBaseURL
		apiKey = config.Providers.OpenAI.APIKey
		baseURL = config.Providers.OpenAI.BaseURL
	case consts.ProviderGoogle:
		inputs[1].Placeholder = "(optional)"
		apiKey = config.Providers.Google.APIKey
		baseURL = config.Providers.Google.BaseURL
	case consts.ProviderAnthropic:
		inputs[1].Placeholder = "(optional)"
		apiKey = config.Providers.Anthropic.APIKey
		baseURL = config.Providers.Anthropic.BaseURL
	}

	if apiKey != "" {
		inputs[0].SetValue(apiKey)
	}
	if baseURL != "" {
		inputs[1].SetValue(baseURL)
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
	inputs := make([]textinput.Model, 3)

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
