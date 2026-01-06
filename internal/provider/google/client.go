package google

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/austiecodes/gomor/internal/client"
	"github.com/austiecodes/gomor/internal/types"
	"google.golang.org/genai"
)

// Message is a thin wrapper around genai.Content.
type Message genai.Content

// compile time check
var _ client.Message = (*Message)(nil)

func (m *Message) GetRole() string {
	if m == nil {
		return ""
	}
	return string(m.Role)
}

func (m *Message) GetContent() any {
	if m == nil {
		return ""
	}
	if len(m.Parts) > 0 {
		return m.Parts[0].Text
	}
	return ""
}

// UserMessage creates a user message.
func UserMessage(content string) Message {
	return Message{
		Role: "user",
		Parts: []*genai.Part{
			{Text: content},
		},
	}
}

// AssistantMessage creates an assistant message.
func AssistantMessage(content string) Message {
	return Message{
		Role: "model",
		Parts: []*genai.Part{
			{Text: content},
		},
	}
}

// SystemMessage creates a system message.
func SystemMessage(content string) Message {
	return Message{
		Role: "system",
		Parts: []*genai.Part{
			{Text: content},
		},
	}
}

// ChatRequest wraps Gemini request.
type ChatRequest struct {
	Model    string
	Messages []Message
	Config   *genai.GenerateContentConfig
}

var _ client.ChatRequest = (*ChatRequest)(nil)

func (r *ChatRequest) GetModel() types.Model {
	return types.Model{Provider: "google", ModelID: r.Model}
}

func NewChatRequest(modelID string) *ChatRequest {
	return &ChatRequest{
		Model:  modelID,
		Config: &genai.GenerateContentConfig{},
	}
}

func (r *ChatRequest) WithMessages(msgs ...Message) *ChatRequest {
	r.Messages = append(r.Messages, msgs...)
	return r
}

func (r *ChatRequest) WithTemperature(t float64) *ChatRequest {
	f32 := float32(t)
	r.Config.Temperature = &f32
	return r
}

// ChatResponse implements client.ChatResponse
type ChatResponse struct {
	*genai.GenerateContentResponse
}

func (r *ChatResponse) GetContent() any {
	if len(r.Candidates) > 0 && len(r.Candidates[0].Content.Parts) > 0 {
		return r.Candidates[0].Content.Parts[0].Text
	}
	return ""
}

// StreamResponse implements client.StreamResponse
type StreamResponse struct {
	next    func() (*genai.GenerateContentResponse, error, bool)
	stop    func()
	current *genai.GenerateContentResponse
	err     error
}

func (s *StreamResponse) Next() bool {
	resp, err, ok := s.next()
	if !ok {
		return false
	}
	if err != nil {
		s.err = err
		return false
	}
	s.current = resp
	return true
}

func (s *StreamResponse) GetChunk() string {
	if s.current != nil && len(s.current.Candidates) > 0 && len(s.current.Candidates[0].Content.Parts) > 0 {
		return s.current.Candidates[0].Content.Parts[0].Text
	}
	return ""
}

func (s *StreamResponse) Err() error {
	return s.err
}

func (s *StreamResponse) Close() error {
	if s.stop != nil {
		s.stop()
	}
	return nil
}

type Client struct {
	client *genai.Client
}

func NewClient(apiKey, baseURL string) *Client {
	ctx := context.Background()
	cfg := &genai.ClientConfig{
		APIKey: apiKey,
	}
	c, err := genai.NewClient(ctx, cfg)
	if err != nil {
		// For consistency with OpenAI's NewClient (which doesn't return error in the struct),
		// we might need to handle this. But factory expects NewQueryClient to handle errors.
		// We'll see how factory uses it.
		return nil
	}
	return &Client{client: c}
}

func (c *Client) Chat(ctx context.Context, request *ChatRequest) (client.ChatResponse, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("google client not initialized")
	}

	contents, systemInstruction := c.prepareContents(request.Messages)
	if systemInstruction != nil {
		request.Config.SystemInstruction = systemInstruction
	}

	resp, err := c.client.Models.GenerateContent(ctx, request.Model, contents, request.Config)
	if err != nil {
		return nil, err
	}
	return &ChatResponse{GenerateContentResponse: resp}, nil
}

func (c *Client) ChatStream(ctx context.Context, request *ChatRequest) (client.StreamResponse, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("google client not initialized")
	}

	contents, systemInstruction := c.prepareContents(request.Messages)
	if systemInstruction != nil {
		request.Config.SystemInstruction = systemInstruction
	}

	stream := c.client.Models.GenerateContentStream(ctx, request.Model, contents, request.Config)
	next, stop := iter.Pull2(stream)

	return &StreamResponse{
		next: next,
		stop: stop,
	}, nil
}

func (c *Client) prepareContents(msgs []Message) ([]*genai.Content, *genai.Content) {
	var contents []*genai.Content
	var systemInstruction *genai.Content

	for _, m := range msgs {
		if m.Role == "system" {
			systemInstruction = &genai.Content{
				Parts: m.Parts,
			}
		} else {
			contents = append(contents, (*genai.Content)(&m))
		}
	}
	return contents, systemInstruction
}

func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("google client not initialized")
	}

	page, err := c.client.Models.List(ctx, nil)
	if err != nil {
		return nil, err
	}
	var models []string
	for _, m := range page.Items {
		models = append(models, strings.TrimPrefix(m.Name, "models/"))
	}
	return models, nil
}
