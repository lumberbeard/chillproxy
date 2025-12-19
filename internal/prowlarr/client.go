package prowlarr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type SearchResult struct {
	GUID        string `json:"guid"`
	Title       string `json:"title"`
	InfoHash    string `json:"infoHash"`
	Indexer     string `json:"indexer"`
	Seeders     int    `json:"seeders"`
	Leechers    int    `json:"leechers"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"downloadUrl"`
	PublishDate string `json:"publishDate"`
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Search queries Prowlarr for torrents matching the query
func (c *Client) Search(ctx context.Context, query string, searchType string) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("type", searchType)

	req, _ := http.NewRequestWithContext(ctx, "GET",
		c.baseURL+"/api/v1/search?"+params.Encode(), nil)
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prowlarr search failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prowlarr returned %d", resp.StatusCode)
	}

	var results []SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	return results, nil
}

// GetInstance returns a singleton Prowlarr client or nil if not configured
func GetInstance() *Client {
	if !IsConfigured() {
		return nil
	}
	return NewClient(URL, APIKey)
}

