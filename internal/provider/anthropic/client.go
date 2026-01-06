package anthropic

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/austiecodes/gomor/internal/client"
	"github.com/austiecodes/gomor/internal/types"
)

// Message is a wrapper around Anthropic's MessageParam.
// It implements client.Message for cross-provider abstraction.
type Message anthropic.MessageParam

// compile time check that Message implements client.Message
var _ client.Message = (*Message)(nil)

func (m *Message) GetRole() string {
	if m == nil {
		return ""
	}
	p := anthropic.MessageParam(*m)
	return string(p.Role)
}

func (m *Message) GetContent() any {
	if m == nil {
		return ""
	}
	p := anthropic.MessageParam(*m)
	return p.Content
}

// UserMessage creates a user message.
func UserMessage(content string) Message {
	return Message(anthropic.NewUserMessage(anthropic.NewTextBlock(content)))
}

// AssistantMessage creates an assistant message.
func AssistantMessage(content string) Message {
	return Message(anthropic.NewAssistantMessage(anthropic.NewTextBlock(content)))
}

// SystemMessage creates a system message.
// Note: Anthropic uses a top-level "system" parameter in requests, not a message role.
// This helper is for compatibility, but the QueryClient should handle it separately if possible.
// However, for strict interface compliance, we might need to handle it.
// Anthropic's MessageParam DOES NOT support "system" role.
// So we'll return a user message with a special prefix or handle it at the request level.
// Ideally, we shouldn't use SystemMessage here for Anthropic in the message list.
// But for now, let's just panic or return a placeholder if used incorrectly,
// or better, allow it and let the request builder extract it.
// Actually, `client.Message` interface just needs GetRole/GetContent.
// We can define a custom struct for SystemMessage if needed, but for now let's see how it's used.
// The `query_client.go` usually separates system context.
// Let's implement it as a UserMessage for now to verify interface, but we might need a different approach.
// modifying the approach: we won't strictly use Anthropic's types for *storage* if we need a system message.
// But wait, `QueryClient` in `openai/query_client.go` separates `systemContext`.
// So we might not need `SystemMessage` here if we use `QueryClient` correctly.
func SystemMessage(content string) Message {
	// Anthropic treats system prompts as a separate parameter, not a message in the messages list.
	// We'll return a placeholder that shouldn't be sent in the messages list.
	// Or we can just return a UserMessage and rely on the caller to not use it as system.
	// A better way is to not support SystemMessage in the *message list* for Anthropic.
	return Message(anthropic.NewUserMessage(anthropic.NewTextBlock("SYSTEM: " + content)))
}

// ChatRequest wraps Anthropic's MessageNewParams and implements client.ChatRequest.
type ChatRequest struct {
	anthropic.MessageNewParams
}

// compile time check that ChatRequest implements client.ChatRequest
var _ client.ChatRequest = (*ChatRequest)(nil)

func (r *ChatRequest) GetModel() types.Model {
	if r == nil {
		return types.Model{Provider: "anthropic", ModelID: ""}
	}
	// Model matches, assuming it can be cast to string or has String()
	return types.Model{Provider: "anthropic", ModelID: string(r.Model)}
}

// NewChatRequest creates a new chat request with the given model ID.
func NewChatRequest(modelID string) *ChatRequest {
	params := anthropic.MessageNewParams{
		Model: anthropic.Model(modelID),
	}
	// Try to set MaxTokens directly if Int() isn't working as field
	// Based on error: cannot use anthropic.Int(1024) (value of struct type param.Opt[int64]) as int64
	// So fields are raw types.
	params.MaxTokens = 1024
	return &ChatRequest{MessageNewParams: params}
}

// WithMessages sets the request messages.
func (r *ChatRequest) WithMessages(msgs ...Message) *ChatRequest {
	r.Messages = make([]anthropic.MessageParam, len(msgs))
	for i, m := range msgs {
		r.Messages[i] = anthropic.MessageParam(m)
	}
	return r
}

// WithSystem sets the system prompt.
func (r *ChatRequest) WithSystem(system string) *ChatRequest {
	if system != "" {
		// Manually construct TextBlockParam as we can't find helper/constant easily
		// and System field requires []TextBlockParam.
		r.System = []anthropic.TextBlockParam{
			{
				Text: system,
				Type: "text", // Implicit cast hoping constant.Text accepts string
			},
		}
	}
	return r
}

// ChatResponse embeds Anthropic response and implements client.ChatResponse
type ChatResponse struct {
	*anthropic.Message
}

func (r *ChatResponse) GetContent() any {
	if len(r.Content) > 0 {
		return r.Content[0].Text
	}
	return ""
}

// StreamResponse embeds Anthropic stream and implements client.StreamResponse
type StreamResponse struct {
	stream  *ssestream.Stream[anthropic.MessageStreamEventUnion]
	current anthropic.MessageStreamEventUnion
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
	if s.current.Type == "content_block_delta" {
		// Use helper to cast safely
		evt := s.current.AsContentBlockDelta()
		return evt.Delta.Text
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

// Client is an Anthropic API client
type Client struct {
	client *anthropic.Client
}

// NewClient creates a new Anthropic client
func NewClient(apiKey, baseURL string) *Client {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	c := anthropic.NewClient(opts...)
	return &Client{client: &c}
}

// Chat calls the Anthropic Messages API.
func (c *Client) Chat(ctx context.Context, request *ChatRequest) (client.ChatResponse, error) {
	resp, err := c.client.Messages.New(ctx, request.MessageNewParams)
	if err != nil {
		return nil, err
	}

	return &ChatResponse{Message: resp}, nil
}

// ChatStream calls the Anthropic Messages API in streaming mode.
func (c *Client) ChatStream(ctx context.Context, request *ChatRequest) (client.StreamResponse, error) {
	stream := c.client.Messages.NewStreaming(ctx, request.MessageNewParams)

	return &StreamResponse{stream: stream}, nil
}

// ListModels fetches available models from the Anthropic API
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	page, err := c.client.Models.List(ctx, anthropic.ModelListParams{})
	if err != nil {
		return nil, err
	}

	var models []string
	for _, m := range page.Data {
		models = append(models, m.ID)
	}

	return models, nil
}
