package google

import (
	"context"
	"fmt"

	"github.com/austiecodes/gomor/internal/client"
	"github.com/austiecodes/gomor/internal/types"
)

type QueryClient struct {
	c *Client
}

func NewQueryClient(apiKey, baseURL string) *QueryClient {
	return &QueryClient{c: NewClient(apiKey, baseURL)}
}

func (q *QueryClient) ChatStream(ctx context.Context, model types.Model, query string) (client.StreamResponse, error) {
	if q.c == nil {
		return nil, fmt.Errorf("google client not initialized")
	}
	req := NewChatRequest(model.ModelID).WithMessages(UserMessage(query))
	return q.c.ChatStream(ctx, req)
}

func (q *QueryClient) ChatStreamWithContext(ctx context.Context, model types.Model, systemContext, query string) (client.StreamResponse, error) {
	if q.c == nil {
		return nil, fmt.Errorf("google client not initialized")
	}
	var msgs []Message
	if systemContext != "" {
		msgs = append(msgs, SystemMessage(systemContext))
	}
	msgs = append(msgs, UserMessage(query))
	req := NewChatRequest(model.ModelID).WithMessages(msgs...)
	return q.c.ChatStream(ctx, req)
}

func (q *QueryClient) ListModels(ctx context.Context) ([]string, error) {
	if q.c == nil {
		return nil, fmt.Errorf("google client not initialized")
	}
	return q.c.ListModels(ctx)
}
