package chillstreams

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://test.com", "test-key")

	if client.baseURL != "http://test.com" {
		t.Errorf("Expected baseURL http://test.com, got %s", client.baseURL)
	}

	if client.apiKey != "test-key" {
		t.Errorf("Expected apiKey test-key, got %s", client.apiKey)
	}

	if client.client.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", client.client.Timeout)
	}
}

func TestGetPoolKey_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/api/v1/internal/pool/get-key" {
			t.Errorf("Expected /api/v1/internal/pool/get-key, got %s", r.URL.Path)
		}

		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Parse request body
		var req GetPoolKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if req.UserID != "test-user" {
			t.Errorf("Expected UserID test-user, got %s", req.UserID)
		}

		// Send response
		response := GetPoolKeyResponse{
			PoolKey:     "pool-key-123",
			PoolKeyID:   "key-id-456",
			Allowed:     true,
			DeviceCount: 2,
			Message:     "",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Test client
	client := NewClient(server.URL, "test-key")

	req := GetPoolKeyRequest{
		UserID:   "test-user",
		DeviceID: "device-123",
		Action:   "check-cache",
		Hash:     "torrent-hash",
	}

	resp, err := client.GetPoolKey(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.PoolKey != "pool-key-123" {
		t.Errorf("Expected PoolKey pool-key-123, got %s", resp.PoolKey)
	}

	if resp.PoolKeyID != "key-id-456" {
		t.Errorf("Expected PoolKeyID key-id-456, got %s", resp.PoolKeyID)
	}

	if !resp.Allowed {
		t.Errorf("Expected Allowed true, got false")
	}

	if resp.DeviceCount != 2 {
		t.Errorf("Expected DeviceCount 2, got %d", resp.DeviceCount)
	}
}

func TestGetPoolKey_NotAllowed(t *testing.T) {
	// Mock server returning not allowed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := GetPoolKeyResponse{
			Allowed: false,
			Message: "Maximum device limit reached",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")

	req := GetPoolKeyRequest{
		UserID:   "test-user",
		DeviceID: "new-device",
		Action:   "check-cache",
	}

	resp, err := client.GetPoolKey(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.Allowed {
		t.Errorf("Expected Allowed false, got true")
	}

	if resp.Message != "Maximum device limit reached" {
		t.Errorf("Expected message about device limit, got %s", resp.Message)
	}
}

func TestGetPoolKey_ServerError(t *testing.T) {
	// Mock server returning error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")

	req := GetPoolKeyRequest{
		UserID:   "test-user",
		DeviceID: "device-123",
		Action:   "check-cache",
	}

	_, err := client.GetPoolKey(context.Background(), req)

	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestGetPoolKey_Unauthorized(t *testing.T) {
	// Mock server checking auth
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer correct-key" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "wrong-key")

	req := GetPoolKeyRequest{
		UserID:   "test-user",
		DeviceID: "device-123",
		Action:   "check-cache",
	}

	_, err := client.GetPoolKey(context.Background(), req)

	if err == nil {
		t.Errorf("Expected error for unauthorized, got nil")
	}
}

func TestLogUsage_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/api/v1/internal/pool/log-usage" {
			t.Errorf("Expected /api/v1/internal/pool/log-usage, got %s", r.URL.Path)
		}

		// Parse request body
		var req LogUsageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if req.UserID != "test-user" {
			t.Errorf("Expected UserID test-user, got %s", req.UserID)
		}

		if req.Action != "stream-served" {
			t.Errorf("Expected Action stream-served, got %s", req.Action)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")

	req := LogUsageRequest{
		UserID:    "test-user",
		PoolKeyID: "key-123",
		Action:    "stream-served",
		Hash:      "torrent-hash",
		Cached:    true,
		Bytes:     1500000000,
	}

	err := client.LogUsage(context.Background(), req)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestLogUsage_Error(t *testing.T) {
	// Mock server returning error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")

	req := LogUsageRequest{
		UserID: "test-user",
		Action: "stream-served",
	}

	err := client.LogUsage(context.Background(), req)

	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestGetPoolKey_Timeout(t *testing.T) {
	// Mock server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(15 * time.Second) // Longer than client timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")

	req := GetPoolKeyRequest{
		UserID:   "test-user",
		DeviceID: "device-123",
		Action:   "check-cache",
	}

	_, err := client.GetPoolKey(context.Background(), req)

	if err == nil {
		t.Errorf("Expected timeout error, got nil")
	}
}

