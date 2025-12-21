package memory

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"

	memstore "github.com/austiecodes/goa/internal/memory"
)

// Screen represents the current TUI screen
type Screen int

const (
	ScreenMemoryList Screen = iota
	ScreenMemoryDetail
	ScreenMemoryAdd
	ScreenMemoryEdit
	ScreenConfirmDelete
)

// MemoryListItem implements list.Item interface for memory display
type MemoryListItem struct {
	Memory memstore.MemoryItem
}

func (i MemoryListItem) Title() string       { return i.Memory.Text }
func (i MemoryListItem) Description() string { return i.Memory.CreatedAt.Format("2006-01-02 15:04") }
func (i MemoryListItem) FilterValue() string { return i.Memory.Text }

// Model is the Bubble Tea model for the memory command
type Model struct {
	Screen         Screen
	List           list.Model
	Viewport       viewport.Model
	TextInputs     []textinput.Model
	FocusedInput   int
	SelectedMemory *memstore.MemoryItem
	Memories       []memstore.MemoryItem
	Err            error
	StatusMsg      string
	Quitting       bool
	Width          int
	Height         int
}

// MemoriesLoadedMsg is sent when memories are loaded from store
type MemoriesLoadedMsg struct {
	Memories []memstore.MemoryItem
	Err      error
}

// MemorySavedMsg is sent when a memory is saved
type MemorySavedMsg struct {
	Err error
}

// MemoryDeletedMsg is sent when a memory is deleted
type MemoryDeletedMsg struct {
	Err error
}

