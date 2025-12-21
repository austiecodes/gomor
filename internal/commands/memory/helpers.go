package memory

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	memstore "github.com/austiecodes/goa/internal/memory"
	"github.com/austiecodes/goa/internal/provider"
	"github.com/austiecodes/goa/internal/utils"
)

func createMemoryList(memories []memstore.MemoryItem, width, height int) list.Model {
	items := make([]list.Item, len(memories))
	for i, mem := range memories {
		items[i] = MemoryListItem{Memory: mem}
	}

	delegate := list.NewDefaultDelegate()
	w := min(width-4, 80)
	h := min(height-6, 20)
	if w < 40 {
		w = 40
	}
	if h < 10 {
		h = 10
	}

	l := list.New(items, delegate, w, h)
	l.Title = "Memories"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
			key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		}
	}
	return l
}

func createAddEditInputs(mem *memstore.MemoryItem) []textinput.Model {
	inputs := make([]textinput.Model, 2)

	// Text input
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Enter preference or fact..."
	inputs[0].CharLimit = 500
	inputs[0].Width = 60
	if mem != nil {
		inputs[0].SetValue(mem.Text)
	}

	// Tags input
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "tag1, tag2, tag3 (optional)"
	inputs[1].CharLimit = 200
	inputs[1].Width = 60
	if mem != nil && len(mem.Tags) > 0 {
		inputs[1].SetValue(strings.Join(mem.Tags, ", "))
	}

	return inputs
}

func loadMemories() tea.Cmd {
	return func() tea.Msg {
		store, err := memstore.NewStore()
		if err != nil {
			return MemoriesLoadedMsg{Err: err}
		}
		defer store.Close()

		memories, err := store.GetAllMemories()
		return MemoriesLoadedMsg{Memories: memories, Err: err}
	}
}

func saveNewMemory(text string, tags []string) tea.Cmd {
	return func() tea.Msg {
		config, err := utils.LoadConfig()
		if err != nil {
			return MemorySavedMsg{Err: err}
		}

		if config.Model.EmbeddingModel == nil {
			return MemorySavedMsg{Err: err}
		}

		// Create embedding client
		embeddingModel := *config.Model.EmbeddingModel
		embClient, err := provider.NewEmbeddingClient(config, embeddingModel.Provider)
		if err != nil {
			return MemorySavedMsg{Err: err}
		}

		// Generate embedding
		ctx := context.Background()
		embedding, err := embClient.Embed(ctx, embeddingModel, text)
		if err != nil {
			return MemorySavedMsg{Err: err}
		}

		// Normalize embedding
		normalizedEmbedding := memstore.NormalizeVector(embedding)

		// Open store and save
		store, err := memstore.NewStore()
		if err != nil {
			return MemorySavedMsg{Err: err}
		}
		defer store.Close()

		item := &memstore.MemoryItem{
			Text:       text,
			Tags:       tags,
			Source:     memstore.SourceExplicit,
			Confidence: 1.0,
			Provider:   embeddingModel.Provider,
			ModelID:    embeddingModel.ModelID,
			Dim:        len(normalizedEmbedding),
			Embedding:  normalizedEmbedding,
		}

		err = store.SaveMemory(item)
		return MemorySavedMsg{Err: err}
	}
}

func updateMemory(id, text string, tags []string) tea.Cmd {
	return func() tea.Msg {
		config, err := utils.LoadConfig()
		if err != nil {
			return MemorySavedMsg{Err: err}
		}

		if config.Model.EmbeddingModel == nil {
			return MemorySavedMsg{Err: err}
		}

		// Create embedding client
		embeddingModel := *config.Model.EmbeddingModel
		embClient, err := provider.NewEmbeddingClient(config, embeddingModel.Provider)
		if err != nil {
			return MemorySavedMsg{Err: err}
		}

		// Generate new embedding
		ctx := context.Background()
		embedding, err := embClient.Embed(ctx, embeddingModel, text)
		if err != nil {
			return MemorySavedMsg{Err: err}
		}

		// Normalize embedding
		normalizedEmbedding := memstore.NormalizeVector(embedding)

		// Open store
		store, err := memstore.NewStore()
		if err != nil {
			return MemorySavedMsg{Err: err}
		}
		defer store.Close()

		// Delete old and save new (simple update strategy)
		_ = store.DeleteMemory(id)

		item := &memstore.MemoryItem{
			Text:       text,
			Tags:       tags,
			Source:     memstore.SourceExplicit,
			Confidence: 1.0,
			Provider:   embeddingModel.Provider,
			ModelID:    embeddingModel.ModelID,
			Dim:        len(normalizedEmbedding),
			Embedding:  normalizedEmbedding,
		}

		err = store.SaveMemory(item)
		return MemorySavedMsg{Err: err}
	}
}

func deleteMemory(id string) tea.Cmd {
	return func() tea.Msg {
		store, err := memstore.NewStore()
		if err != nil {
			return MemoryDeletedMsg{Err: err}
		}
		defer store.Close()

		err = store.DeleteMemory(id)
		return MemoryDeletedMsg{Err: err}
	}
}

func parseTags(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}

	parts := strings.Split(input, ",")
	var tags []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			tags = append(tags, p)
		}
	}
	return tags
}

