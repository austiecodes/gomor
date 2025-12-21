package set

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/austiecodes/goa/internal/utils"
)

func initialModel() Model {
	config, err := utils.LoadConfig()
	if err != nil {
		config = utils.DefaultConfig()
	}

	l := createMainMenu()

	return Model{
		Screen: ScreenMainMenu,
		Config: config,
		List:   l,
		Err:    err,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.List.SetSize(min(msg.Width-4, 80), min(msg.Height-4, 30))
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.Screen == ScreenMainMenu {
				m.Quitting = true
				return m, tea.Quit
			}
			// Go back to main menu
			m.Screen = ScreenMainMenu
			m.List = createMainMenu()
			m.Err = nil
			return m, nil

		case "esc":
			if m.Screen != ScreenMainMenu {
				m.Screen = ScreenMainMenu
				m.List = createMainMenu()
				m.Err = nil
				return m, nil
			}
		}

	case ModelsLoadedMsg:
		if msg.Err != nil {
			m.Err = msg.Err
			m.Screen = ScreenMainMenu
			m.List = createMainMenu()
			return m, nil
		}
		m.List = createModelList(msg.Models, m.ModelType)
		m.Screen = ScreenModelSelect
		return m, nil

	case ConfigSavedMsg:
		if msg.Err != nil {
			m.Err = msg.Err
		} else {
			m.Screen = ScreenMainMenu
			m.List = createMainMenu()
		}
		return m, nil
	}

	switch m.Screen {
	case ScreenMainMenu:
		return m.updateMainMenu(msg)
	case ScreenProviderSelect:
		return m.updateProviderSelect(msg)
	case ScreenProviderConfig:
		return m.updateProviderConfig(msg)
	case ScreenModelProviderSelect:
		return m.updateModelProviderSelect(msg)
	case ScreenModelSelect:
		return m.updateModelSelect(msg)
	case ScreenMemoryConfig:
		return m.updateMemoryConfig(msg)
	}

	return m, nil
}

func (m Model) View() string {
	return m.renderView()
}

