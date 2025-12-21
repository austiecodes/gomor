package memory

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func initialModel() Model {
	// Create an empty list initially, will be populated after load
	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 60, 14)
	l.Title = "Memories"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)

	return Model{
		Screen:    ScreenMemoryList,
		List:      l,
		StatusMsg: "Loading memories...",
	}
}

func (m Model) Init() tea.Cmd {
	return loadMemories()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.List.SetSize(min(msg.Width-4, 80), min(msg.Height-6, 20))
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.Screen == ScreenMemoryList {
				m.Quitting = true
				return m, tea.Quit
			}
			// Go back to list
			m.Screen = ScreenMemoryList
			m.SelectedMemory = nil
			m.Err = nil
			m.StatusMsg = ""
			return m, nil

		case "esc":
			if m.Screen != ScreenMemoryList {
				m.Screen = ScreenMemoryList
				m.SelectedMemory = nil
				m.Err = nil
				m.StatusMsg = ""
				return m, nil
			}
		}

	case MemoriesLoadedMsg:
		m.StatusMsg = ""
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		m.Memories = msg.Memories
		m.List = createMemoryList(m.Memories, m.Width, m.Height)
		return m, nil

	case MemorySavedMsg:
		m.StatusMsg = ""
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		// Reload memories and go back to list
		m.Screen = ScreenMemoryList
		m.SelectedMemory = nil
		m.Err = nil
		m.StatusMsg = "Memory saved!"
		return m, loadMemories()

	case MemoryDeletedMsg:
		m.StatusMsg = ""
		if msg.Err != nil {
			m.Err = msg.Err
			return m, nil
		}
		// Reload memories and go back to list
		m.Screen = ScreenMemoryList
		m.SelectedMemory = nil
		m.Err = nil
		m.StatusMsg = "Memory deleted!"
		return m, loadMemories()
	}

	switch m.Screen {
	case ScreenMemoryList:
		return m.updateMemoryList(msg)
	case ScreenMemoryDetail:
		return m.updateMemoryDetail(msg)
	case ScreenMemoryAdd:
		return m.updateMemoryAdd(msg)
	case ScreenMemoryEdit:
		return m.updateMemoryEdit(msg)
	case ScreenConfirmDelete:
		return m.updateConfirmDelete(msg)
	}

	return m, nil
}

func (m Model) View() string {
	return m.renderView()
}

