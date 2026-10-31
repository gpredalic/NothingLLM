package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	apiKey string
	baseURL string
	http   *http.Client
}

func NewClient(apiKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

// Embedding represents a single embedding vector.
type Embedding struct {
	Index int     `json:"index"`
	Value []float32 `json:"embedding"`
}

// Embed creates embeddings for the given texts.
func (c *Client) Embed(ctx context.Context, model string, texts []string) ([][]float32, error) {
	body := map[string]any{
		"model": model,
		"input": texts,
	}
	var resp struct {
		Data []Embedding `json:"data"`
	}
	if err := c.post(ctx, "/embeddings", body, &resp); err != nil {
		return nil, err
	}
	vectors := make([][]float32, len(resp.Data))
	for i, e := range resp.Data {
		vectors[i] = e.Value
	}
	return vectors, nil
}

// ChatMessage represents a single message in a chat conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse is the result of a chat completion.
type ChatResponse struct {
	Content string `json:"content"`
	Model   string `json:"model"`
}

// Chat performs a chat completion.
func (c *Client) Chat(ctx context.Context, model string, messages []ChatMessage, temperature float64, maxTokens int) (*ChatResponse, error) {
	body := map[string]any{
		"model":       model,
		"messages":    messages,
		"temperature": temperature,
	}
	if maxTokens > 0 {
		body["max_tokens"] = maxTokens
	}

	var resp struct {
		Choices []struct {
			Message ChatMessage `json:"message"`
		} `json:"choices"`
		Model string `json:"model"`
	}
	if err := c.post(ctx, "/chat/completions", body, &resp); err != nil {
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai: no choices in response")
	}
	return &ChatResponse{
		Content: resp.Choices[0].Message.Content,
		Model:   resp.Model,
	}, nil
}

// ModelInfo represents an available model.
type ModelInfo struct {
	ID            string  `json:"id"`
	Object        string  `json:"object"`
	Created       int64   `json:"created"`
	ContextWindow int     `json:"context_window"`
}

// ListModels fetches available models from OpenAI.
func (c *Client) ListModels(ctx context.Context) ([]ModelInfo, error) {
	var resp struct {
		Data []ModelInfo `json:"data"`
	}
	if err := c.get(ctx, "/models", &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.errResp(resp)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.setHeaders(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.errResp(resp)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
}

func (c *Client) errResp(resp *http.Response) error {
	data, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("openai: %d: %s", resp.StatusCode, string(data))
}

// Ping checks API connectivity.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("openai: models returned %d", resp.StatusCode)
	}
	return nil
}