package set

import (
	"github.com/austiecodes/gomor/internal/types"
	"github.com/austiecodes/gomor/internal/utils"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
)

// Main Menu Items
const (
	MenuItemProvider       = "provider"
	MenuItemChatModel      = "chat-model"
	MenuItemTitleModel     = "title-model"
	MenuItemThinkModel     = "think-model"
	MenuItemToolModel      = "tool-model"
	MenuItemEmbeddingModel = "embedding-model"
	MenuItemMemory         = "memory"
	MenuItemExit           = "exit"
)

// Screen represents the current TUI screen
type Screen int

const (
	ScreenMainMenu Screen = iota
	ScreenProviderSelect
	ScreenProviderConfig
	ScreenModelProviderSelect
	ScreenModelSelect
	ScreenMemoryConfig
	ScreenConfirmReindex
)

// ModelType represents which model is being configured
type ModelType int

const (
	ModelTypeChat ModelType = iota
	ModelTypeTitle
	ModelTypeThink
	ModelTypeTool
	ModelTypeEmbedding
)

// MenuItem implements list.Item interface
type MenuItem struct {
	title string
	desc  string
}

func (i MenuItem) Title() string       { return i.title }
func (i MenuItem) Description() string { return i.desc }
func (i MenuItem) FilterValue() string { return i.title }

// Model is the Bubble Tea model for the set command
type Model struct {
	Screen           Screen
	Config           *utils.Config
	List             list.Model
	TextInputs       []textinput.Model
	FocusedInput     int
	ModelType        ModelType
	Err              error
	Quitting         bool
	PendingModel     *types.Model
	Reindexing       bool
	Width            int
	Height           int
	SelectedProvider string
}

// ModelsLoadedMsg is sent when models are loaded from API
type ModelsLoadedMsg struct {
	Models []string
	Err    error
}

// ConfigSavedMsg is sent when config is saved
type ConfigSavedMsg struct {
	Err error
}
