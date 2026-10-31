package qdrant

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

type CollectionInfo struct {
	Name       string   `json:"name"`
	Vectors    int64    `json:"vectors_count"`
	Dimensions int      `json:"dimensions"`
	Status     string   `json:"status"`
	PayloadSchema map[string]PayloadField `json:"payload_schema"`
}

type PayloadField struct {
	DataType     string `json:"data_type"`
	Values       int64  `json:"values"`
	SampleValues []any  `json:"sample_values"`
}

// ListCollections returns all collections from Qdrant.
func (c *Client) ListCollections(ctx context.Context) ([]CollectionInfo, error) {
	var resp struct {
		Result struct {
			Collections []struct {
				Name string `json:"name"`
			} `json:"collections"`
		} `json:"result"`
	}
	if err := c.get(ctx, "/collections", &resp); err != nil {
		return nil, err
	}
	var infos []CollectionInfo
	for _, col := range resp.Result.Collections {
		info, err := c.GetCollectionInfo(ctx, col.Name)
		if err != nil {
			// Collection might be in a bad state; skip it
			continue
		}
		infos = append(infos, *info)
	}
	return infos, nil
}

// GetCollectionInfo returns details for a single collection.
func (c *Client) GetCollectionInfo(ctx context.Context, name string) (*CollectionInfo, error) {
	var resp struct {
		Result struct {
			PointsCount *int64 `json:"points_count"`
			Status      string `json:"status"`
			Config      struct {
				Params struct {
					Vectors struct {
						Size int `json:"size"`
					} `json:"vectors"`
				} `json:"params"`
			} `json:"config"`
			PayloadSchema map[string]PayloadField `json:"payload_schema"`
		} `json:"result"`
	}
	path := "/collections/" + name
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, err
	}

	dims := 0
	if resp.Result.Config.Params.Vectors.Size > 0 {
		dims = resp.Result.Config.Params.Vectors.Size
	}
	vectors := int64(0)
	if resp.Result.PointsCount != nil {
		vectors = *resp.Result.PointsCount
	}

	return &CollectionInfo{
		Name:          name,
		Vectors:       vectors,
		Dimensions:    dims,
		Status:        resp.Result.Status,
		PayloadSchema: resp.Result.PayloadSchema,
	}, nil
}

// Search performs a vector search on a collection.
func (c *Client) Search(ctx context.Context, collection string, vector []float32, topK int, filter any) ([]SearchResult, error) {
	body := map[string]any{
		"vector": vector,
		"limit":  topK,
		"with_payload": true,
	}
	if filter != nil {
		body["filter"] = filter
	}

	var resp struct {
		Result struct {
			Points []struct {
				ID      any             `json:"id"`
				Score   float64         `json:"score"`
				Payload map[string]any  `json:"payload"`
			} `json:"points"`
		} `json:"result"`
	}
	path := "/collections/" + collection + "/points/search"
	if err := c.post(ctx, path, body, &resp); err != nil {
		return nil, err
	}
	var results []SearchResult
	for _, p := range resp.Result.Points {
		results = append(results, SearchResult{
			ID:      fmt.Sprintf("%v", p.ID),
			Score:   p.Score,
			Payload: p.Payload,
		})
	}
	return results, nil
}

type SearchResult struct {
	ID      string         `json:"id"`
	Score   float64        `json:"score"`
	Payload map[string]any `json:"payload"`
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
		return c.errFromResp(resp)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, strings.NewReader(string(data)))
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
		return c.errFromResp(resp)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) setHeaders(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}
}

func (c *Client) errFromResp(resp *http.Response) error {
	data, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("qdrant: %d: %s", resp.StatusCode, string(data))
}

// Ping checks if Qdrant is reachable.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qdrant: healthz returned %d", resp.StatusCode)
	}
	return nil
}