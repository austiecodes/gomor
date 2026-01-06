package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/austiecodes/gomor/internal/provider"
	"github.com/austiecodes/gomor/internal/utils"
)

var rootCmd = &cobra.Command{
	Use:   "gomor",
	Short: "gomor is a command-line tool for interacting with LLM APIs",
	Long:  `gomor is a command-line tool for interacting with LLM APIs.`,
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}

		query := strings.Join(args, " ")
		if err := runQuery(query); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

// AddCommand adds a subcommand to the root command
func AddCommand(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runQuery(query string) error {
	config, err := utils.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if config.Model.ChatModel == nil {
		return fmt.Errorf("chat model not configured. Run 'gomor set' to configure your model")
	}

	model := *config.Model.ChatModel
	providerName := model.Provider

	// Create provider client
	c, err := provider.NewQueryClient(config, providerName)
	if err != nil {
		return fmt.Errorf("failed to create query client: %w", err)
	}

	// Call the provider with streaming
	stream, err := c.ChatStream(context.Background(), model, query)
	if err != nil {
		return fmt.Errorf("failed to start chat stream: %w", err)
	}
	defer stream.Close()

	// Output chunks in real-time
	for stream.Next() {
		chunk := stream.GetChunk()
		fmt.Fprint(os.Stdout, chunk)
		os.Stdout.Sync()
	}
	if err := stream.Err(); err != nil {
		return fmt.Errorf("stream error: %w", err)
	}

	fmt.Println()
	return nil
}
