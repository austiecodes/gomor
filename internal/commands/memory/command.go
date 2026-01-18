package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	memoryservice "github.com/austiecodes/gomor/internal/memory/service"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	saveMemoryFn         = memoryservice.Save
	queryMemoryFn        = memoryservice.Retrieve
	deleteMemoryFn       = memoryservice.Delete
	runInteractiveMemory = func() error {
		p := tea.NewProgram(initialModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running memory manager: %w", err)
		}
		return nil
	}
)

type memoryCommandOptions struct {
	saveText   string
	queryText  string
	deleteID   string
	tags       string
	jsonOutput bool
}

type memoryQueryMatch struct {
	ID     string   `json:"id"`
	Text   string   `json:"text"`
	Tags   []string `json:"tags,omitempty"`
	Score  float64  `json:"score"`
	Source string   `json:"source"`
}

type memorySaveOutput struct {
	Message string `json:"message"`
	ID      string `json:"id"`
}

type memoryQueryOutput struct {
	Results string             `json:"results"`
	Matches []memoryQueryMatch `json:"matches,omitempty"`
}

type memoryDeleteOutput struct {
	Message string `json:"message"`
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

var MemoryCmd = newMemoryCommand()

func newMemoryCommand() *cobra.Command {
	opts := &memoryCommandOptions{}

	cmd := &cobra.Command{
		Use:          "memory",
		Short:        "Manage memories interactively",
		Long:         `Open an interactive TUI to view, add, edit, and delete stored memories.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMemoryCommand(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.saveText, "save", "", "save a memory without opening the TUI")
	cmd.Flags().StringVar(&opts.queryText, "query", "", "retrieve memories without opening the TUI")
	cmd.Flags().StringVar(&opts.deleteID, "delete", "", "delete a memory by id without opening the TUI")
	cmd.Flags().StringVar(&opts.tags, "tags", "", "comma-separated tags used with --save")
	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "emit structured JSON output")

	return cmd
}

func runMemoryCommand(cmd *cobra.Command, opts *memoryCommandOptions) error {
	actionCount := countNonEmpty(opts.saveText, opts.queryText, opts.deleteID)
	if actionCount == 0 {
		return runInteractiveMemory()
	}
	if actionCount > 1 {
		return fmt.Errorf("--save, --query, and --delete are mutually exclusive")
	}
	if opts.tags != "" && opts.saveText == "" {
		return fmt.Errorf("--tags can only be used with --save")
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	switch {
	case opts.saveText != "":
		return runSaveCommand(ctx, cmd.OutOrStdout(), opts)
	case opts.queryText != "":
		return runQueryCommand(ctx, cmd.OutOrStdout(), opts)
	default:
		return runDeleteCommand(ctx, cmd.OutOrStdout(), opts)
	}
}

func runSaveCommand(ctx context.Context, out io.Writer, opts *memoryCommandOptions) error {
	result, err := saveMemoryFn(ctx, memoryservice.SaveInput{
		Text: opts.saveText,
		Tags: parseTags(opts.tags),
	})
	if err != nil {
		return err
	}

	output := memorySaveOutput{
		Message: fmt.Sprintf("Memory saved successfully (id: %s)", result.Item.ID),
		ID:      result.Item.ID,
	}

	if opts.jsonOutput {
		return writeJSON(out, output)
	}

	_, err = fmt.Fprintln(out, output.Message)
	return err
}

func runQueryCommand(ctx context.Context, out io.Writer, opts *memoryCommandOptions) error {
	result, err := queryMemoryFn(ctx, memoryservice.RetrieveInput{Query: opts.queryText})
	if err != nil {
		return err
	}

	if opts.jsonOutput {
		return writeJSON(out, memoryQueryOutput{
			Results: result.Text,
			Matches: buildMemoryQueryMatches(result),
		})
	}

	_, err = fmt.Fprintln(out, result.Text)
	return err
}

func runDeleteCommand(ctx context.Context, out io.Writer, opts *memoryCommandOptions) error {
	result, err := deleteMemoryFn(ctx, memoryservice.DeleteInput{ID: opts.deleteID})
	if err != nil {
		return err
	}

	message := fmt.Sprintf("Memory deleted successfully (id: %s)", result.ID)
	if !result.Deleted {
		message = fmt.Sprintf("Memory not found (id: %s)", result.ID)
	}

	output := memoryDeleteOutput{
		Message: message,
		ID:      result.ID,
		Deleted: result.Deleted,
	}

	if opts.jsonOutput {
		return writeJSON(out, output)
	}

	_, err = fmt.Fprintln(out, output.Message)
	return err
}

func buildMemoryQueryMatches(result *memoryservice.RetrieveResult) []memoryQueryMatch {
	if result == nil || result.Response == nil {
		return nil
	}

	matches := make([]memoryQueryMatch, 0, len(result.Response.Results))
	for _, item := range result.Response.Results {
		matches = append(matches, memoryQueryMatch{
			ID:     item.Item.ID,
			Text:   item.Item.Text,
			Tags:   item.Item.Tags,
			Score:  item.Score,
			Source: item.Source,
		})
	}

	return matches
}

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func countNonEmpty(values ...string) int {
	count := 0
	for _, value := range values {
		if value != "" {
			count++
		}
	}
	return count
}
