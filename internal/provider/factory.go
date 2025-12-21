package provider

import (
	"fmt"

	"github.com/austiecodes/goa/internal/client"
	"github.com/austiecodes/goa/internal/consts"
	openaiprov "github.com/austiecodes/goa/internal/provider/openai"
	"github.com/austiecodes/goa/internal/utils"
)

func NewQueryClient(cfg *utils.Config, providerName string) (client.QueryClient, error) {
	switch providerName {
	case consts.ProviderOpenAI:
		openaiCfg := cfg.Providers.OpenAI
		if openaiCfg.APIKey == "" {
			return nil, fmt.Errorf("OpenAI API key not configured. Please configure provider first")
		}
		baseURL := openaiCfg.BaseURL
		if baseURL == "" {
			baseURL = consts.DefaultBaseURL
		}
		return openaiprov.NewQueryClient(openaiCfg.APIKey, baseURL), nil

	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}

// NewEmbeddingClient creates an embedding client for the specified provider.
func NewEmbeddingClient(cfg *utils.Config, providerName string) (client.EmbeddingClient, error) {
	switch providerName {
	case consts.ProviderOpenAI:
		openaiCfg := cfg.Providers.OpenAI
		if openaiCfg.APIKey == "" {
			return nil, fmt.Errorf("OpenAI API key not configured. Please configure provider first")
		}
		baseURL := openaiCfg.BaseURL
		if baseURL == "" {
			baseURL = consts.DefaultBaseURL
		}
		return openaiprov.NewEmbeddingClient(openaiCfg.APIKey, baseURL), nil

	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", providerName)
	}
}
