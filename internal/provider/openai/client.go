package openai

import (
	"context"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/ssestream"

	"github.com/austiecodes/gomor/internal/client"
	"github.com/austiecodes/gomor/internal/types"
)

// Message is a thin wrapper around OpenAI's ChatCompletionMessageParamUnion.
// It implements client.Message for cross-provider abstraction.
type Message openai.ChatCompletionMessageParamUnion

// compile time check that Message implements client.Message
var _ client.Message = (*Message)(nil)

/*
implements client.Message interface below:
*/

func (m *Message) GetRole() string {
	if m == nil {
		return ""
	}
	u := openai.ChatCompletionMessageParamUnion(*m)
	role := u.GetRole()
	if role == nil {
		return ""
	}
	return *role
}

func (m *Message) GetContent() any {
	if m == nil {
		return ""
	}
	u := openai.ChatCompletionMessageParamUnion(*m)
	return u.GetContent()
}

// UserMessage creates a user message.
func UserMessage(content string) Message { return Message(openai.UserMessage(content)) }

// AssistantMessage creates an assistant message.
func AssistantMessage(content string) Message { return Message(openai.AssistantMessage(content)) }

// SystemMessage creates a system message.
func SystemMessage(content string) Message { return Message(openai.SystemMessage(content)) }

// ChatRequest wraps OpenAI's ChatCompletionNewParams and implements provider.ChatRequest.
type ChatRequest openai.ChatCompletionNewParams

// compile time check that ChatRequest implements client.ChatRequest
var _ client.ChatRequest = (*ChatRequest)(nil)

/*
implements client.Message interface below:
*/

func (r *ChatRequest) GetModel() types.Model {
	if r == nil {
		return types.Model{Provider: "openai", ModelID: ""}
	}
	params := openai.ChatCompletionNewParams(*r)
	return types.Model{Provider: "openai", ModelID: string(params.Model)}
}

// NewChatRequest creates a new chat request with the given model ID.
func NewChatRequest(modelID string) *ChatRequest {
	params := openai.ChatCompletionNewParams{
		Model: openai.ChatModel(modelID),
	}
	req := ChatRequest(params)
	return &req
}

// WithMessages sets the request messages.
func (r *ChatRequest) WithMessages(msgs ...Message) *ChatRequest {
	params := openai.ChatCompletionNewParams(*r)
	params.Messages = make([]openai.ChatCompletionMessageParamUnion, len(msgs))
	for i, m := range msgs {
		params.Messages[i] = openai.ChatCompletionMessageParamUnion(m)
	}
	*r = ChatRequest(params)
	return r
}

// WithTemperature sets the request temperature.
func (r *ChatRequest) WithTemperature(t float64) *ChatRequest {
	params := openai.ChatCompletionNewParams(*r)
	params.Temperature = openai.Float(t)
	*r = ChatRequest(params)
	return r
}

// ChatResponse embeds OpenAI response and implements client.ChatResponse
type ChatResponse struct {
	*openai.ChatCompletion
}

func (r *ChatResponse) GetContent() any {
	if len(r.Choices) > 0 {
		return r.Choices[0].Message.Content
	}
	return ""
}

// StreamResponse embeds OpenAI stream and implements client.StreamResponse
type StreamResponse struct {
	stream  *ssestream.Stream[openai.ChatCompletionChunk]
	current openai.ChatCompletionChunk
}

// Next advances to the next chunk
func (s *StreamResponse) Next() bool {
	if s.stream.Next() {
		s.current = s.stream.Current()
		return true
	}
	return false
}

// GetChunk returns the content of the current chunk
func (s *StreamResponse) GetChunk() string {
	if len(s.current.Choices) > 0 {
		return s.current.Choices[0].Delta.Content
	}
	return ""
}

// Err returns any error encountered during iteration
func (s *StreamResponse) Err() error {
	return s.stream.Err()
}

// Close closes the stream
func (s *StreamResponse) Close() error {
	return s.stream.Close()
}

// Client is an OpenAI API client
type Client struct {
	client openai.Client
}

// NewClient creates a new OpenAI client
func NewClient(apiKey, baseURL string) *Client {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	return &Client{client: openai.NewClient(opts...)}
}

// Chat calls the OpenAI Chat Completions API.
func (c *Client) Chat(ctx context.Context, request *ChatRequest) (client.ChatResponse, error) {
	params := openai.ChatCompletionNewParams(*request)
	resp, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}

	return &ChatResponse{ChatCompletion: resp}, nil
}

// ChatStream calls the OpenAI Chat Completions API in streaming mode.
func (c *Client) ChatStream(ctx context.Context, request *ChatRequest) (client.StreamResponse, error) {
	params := openai.ChatCompletionNewParams(*request)
	stream := c.client.Chat.Completions.NewStreaming(ctx, params)

	return &StreamResponse{stream: stream}, nil
}

// ListModels fetches available models from the OpenAI API
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	page, err := c.client.Models.List(ctx)
	if err != nil {
		return nil, err
	}

	var models []string
	for _, model := range page.Data {
		models = append(models, model.ID)
	}

	// Sort models for stable ordering
	sortModels(models)
	return models, nil
}

// sortModels sorts models alphabetically
func sortModels(models []string) {
	// A tiny in-place sort to avoid pulling in extra dependencies.
	for i := 0; i < len(models)-1; i++ {
		for j := i + 1; j < len(models); j++ {
			if models[i] > models[j] {
				models[i], models[j] = models[j], models[i]
			}
		}
	}
}
