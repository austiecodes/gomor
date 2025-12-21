package memory

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateMemoryList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if len(m.Memories) == 0 {
				return *m, nil
			}
			selected := m.List.SelectedItem().(MemoryListItem)
			m.SelectedMemory = &selected.Memory
			m.Screen = ScreenMemoryDetail
			return *m, nil

		case "a":
			// Add new memory
			m.TextInputs = createAddEditInputs(nil)
			m.FocusedInput = 0
			m.Screen = ScreenMemoryAdd
			return *m, m.TextInputs[0].Focus()

		case "d":
			// Delete selected memory
			if len(m.Memories) == 0 {
				return *m, nil
			}
			selected := m.List.SelectedItem().(MemoryListItem)
			m.SelectedMemory = &selected.Memory
			m.Screen = ScreenConfirmDelete
			return *m, nil

		case "e":
			// Edit selected memory
			if len(m.Memories) == 0 {
				return *m, nil
			}
			selected := m.List.SelectedItem().(MemoryListItem)
			m.SelectedMemory = &selected.Memory
			m.TextInputs = createAddEditInputs(&selected.Memory)
			m.FocusedInput = 0
			m.Screen = ScreenMemoryEdit
			return *m, m.TextInputs[0].Focus()
		}
	}

	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return *m, cmd
}

func (m *Model) updateMemoryDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "e":
			// Edit this memory
			m.TextInputs = createAddEditInputs(m.SelectedMemory)
			m.FocusedInput = 0
			m.Screen = ScreenMemoryEdit
			return *m, m.TextInputs[0].Focus()

		case "d":
			// Delete this memory
			m.Screen = ScreenConfirmDelete
			return *m, nil
		}
	}

	return *m, nil
}

func (m *Model) updateMemoryAdd(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			text := strings.TrimSpace(m.TextInputs[0].Value())
			if text == "" {
				m.Err = fmt.Errorf("memory text is required")
				return *m, nil
			}

			tags := parseTags(m.TextInputs[1].Value())
			m.StatusMsg = "Saving..."
			return *m, saveNewMemory(text, tags)
		}
	}

	// Update focused text input
	var cmd tea.Cmd
	m.TextInputs[m.FocusedInput], cmd = m.TextInputs[m.FocusedInput].Update(msg)
	return *m, cmd
}

func (m *Model) updateMemoryEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			text := strings.TrimSpace(m.TextInputs[0].Value())
			if text == "" {
				m.Err = fmt.Errorf("memory text is required")
				return *m, nil
			}

			tags := parseTags(m.TextInputs[1].Value())
			m.StatusMsg = "Updating..."
			return *m, updateMemory(m.SelectedMemory.ID, text, tags)
		}
	}

	// Update focused text input
	var cmd tea.Cmd
	m.TextInputs[m.FocusedInput], cmd = m.TextInputs[m.FocusedInput].Update(msg)
	return *m, cmd
}

func (m *Model) updateConfirmDelete(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.StatusMsg = "Deleting..."
			return *m, deleteMemory(m.SelectedMemory.ID)

		case "n", "N", "esc":
			m.Screen = ScreenMemoryList
			m.SelectedMemory = nil
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
	case ScreenMemoryList:
		if len(m.Memories) == 0 {
			s.WriteString(TitleStyle.Render("Memories"))
			s.WriteString("\n\n")
			s.WriteString(SubtitleStyle.Render("No memories stored yet."))
			s.WriteString("\n\n")
			s.WriteString(HelpStyle.Render("Press 'a' to add a new memory, 'q' to quit"))
		} else {
			s.WriteString(m.List.View())
		}

	case ScreenMemoryDetail:
		if m.SelectedMemory != nil {
			s.WriteString(TitleStyle.Render("Memory Detail"))
			s.WriteString("\n\n")

			s.WriteString(DetailLabelStyle.Render("Text:"))
			s.WriteString("\n")
			s.WriteString(DetailValueStyle.Render(m.SelectedMemory.Text))
			s.WriteString("\n\n")

			s.WriteString(DetailLabelStyle.Render("ID:"))
			s.WriteString(" ")
			s.WriteString(DetailValueStyle.Render(m.SelectedMemory.ID))
			s.WriteString("\n\n")

			s.WriteString(DetailLabelStyle.Render("Created:"))
			s.WriteString(" ")
			s.WriteString(DetailValueStyle.Render(m.SelectedMemory.CreatedAt.Format("2006-01-02 15:04:05")))
			s.WriteString("\n\n")

			s.WriteString(DetailLabelStyle.Render("Source:"))
			s.WriteString(" ")
			s.WriteString(DetailValueStyle.Render(string(m.SelectedMemory.Source)))
			s.WriteString("\n\n")

			s.WriteString(DetailLabelStyle.Render("Confidence:"))
			s.WriteString(" ")
			s.WriteString(DetailValueStyle.Render(fmt.Sprintf("%.2f", m.SelectedMemory.Confidence)))
			s.WriteString("\n\n")

			if len(m.SelectedMemory.Tags) > 0 {
				s.WriteString(DetailLabelStyle.Render("Tags:"))
				s.WriteString(" ")
				for i, tag := range m.SelectedMemory.Tags {
					if i > 0 {
						s.WriteString(" ")
					}
					s.WriteString(TagStyle.Render(tag))
				}
				s.WriteString("\n\n")
			}

			s.WriteString(HelpStyle.Render("Press 'e' to edit, 'd' to delete, Esc to go back"))
		}

	case ScreenMemoryAdd:
		s.WriteString(TitleStyle.Render("Add New Memory"))
		s.WriteString("\n\n")
		s.WriteString(InputLabelStyle.Render("Memory Text (required)"))
		s.WriteString("\n")
		s.WriteString(m.TextInputs[0].View())
		s.WriteString("\n\n")
		s.WriteString(InputLabelStyle.Render("Tags (comma-separated, optional)"))
		s.WriteString("\n")
		s.WriteString(m.TextInputs[1].View())
		s.WriteString("\n\n")
		s.WriteString(HelpStyle.Render("Press Enter to save, Esc to cancel, Tab to navigate"))

	case ScreenMemoryEdit:
		s.WriteString(TitleStyle.Render("Edit Memory"))
		s.WriteString("\n\n")
		s.WriteString(InputLabelStyle.Render("Memory Text (required)"))
		s.WriteString("\n")
		s.WriteString(m.TextInputs[0].View())
		s.WriteString("\n\n")
		s.WriteString(InputLabelStyle.Render("Tags (comma-separated, optional)"))
		s.WriteString("\n")
		s.WriteString(m.TextInputs[1].View())
		s.WriteString("\n\n")
		s.WriteString(HelpStyle.Render("Press Enter to save, Esc to cancel, Tab to navigate"))

	case ScreenConfirmDelete:
		s.WriteString(WarningStyle.Render("Confirm Delete"))
		s.WriteString("\n\n")
		s.WriteString("Are you sure you want to delete this memory?\n\n")
		if m.SelectedMemory != nil {
			s.WriteString(DetailValueStyle.Render(m.SelectedMemory.Text))
			s.WriteString("\n\n")
		}
		s.WriteString(HelpStyle.Render("Press 'y' to confirm, 'n' or Esc to cancel"))
	}

	if m.StatusMsg != "" {
		s.WriteString("\n\n")
		s.WriteString(SubtitleStyle.Render(m.StatusMsg))
	}

	if m.Err != nil {
		s.WriteString("\n\n")
		s.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.Err)))
	}

	return s.String()
}

