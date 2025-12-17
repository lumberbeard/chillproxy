package chillstreams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// GetPoolKey fetches assigned pool key for user
func (c *Client) GetPoolKey(ctx context.Context, req GetPoolKeyRequest) (*GetPoolKeyResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/internal/pool/get-key", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call chillstreams: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chillstreams returned %d", resp.StatusCode)
	}

	var result GetPoolKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// LogUsage logs pool key usage to Chillstreams
func (c *Client) LogUsage(ctx context.Context, req LogUsageRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/internal/pool/log-usage", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call chillstreams: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("log usage failed: %d", resp.StatusCode)
	}

	return nil
}

type GetPoolKeyRequest struct {
	UserID   string `json:"userId"`
	DeviceID string `json:"deviceId"`
	Action   string `json:"action"`
	Hash     string `json:"hash,omitempty"`
}

type GetPoolKeyResponse struct {
	PoolKey     string `json:"poolKey"`
	PoolKeyID   string `json:"poolKeyId,omitempty"`
	Allowed     bool   `json:"allowed"`
	DeviceCount int    `json:"deviceCount"`
	Message     string `json:"message,omitempty"`
}

type LogUsageRequest struct {
	UserID    string `json:"userId"`
	PoolKeyID string `json:"poolKeyId,omitempty"`
	Action    string `json:"action"`
	Hash      string `json:"hash,omitempty"`
	Cached    bool   `json:"cached,omitempty"`
	Bytes     int64  `json:"bytes,omitempty"`
}

