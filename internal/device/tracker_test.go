package device

import (
	"net/http"
	"testing"
)

func TestGenerateDeviceID_Consistency(t *testing.T) {
	// Create mock request
	req1, _ := http.NewRequest("GET", "http://example.com", nil)
	req1.RemoteAddr = "192.168.1.100:12345"
	req1.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	req2, _ := http.NewRequest("GET", "http://example.com", nil)
	req2.RemoteAddr = "192.168.1.100:54321" // Different port
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	// Same IP and UA should generate same device ID
	id1 := GenerateDeviceID(req1)
	id2 := GenerateDeviceID(req2)

	if id1 != id2 {
		t.Errorf("Expected same device ID for same IP+UA, got %s and %s", id1, id2)
	}

	// Should be 64 character hex string (SHA256)
	if len(id1) != 64 {
		t.Errorf("Expected 64 character device ID, got %d", len(id1))
	}
}

func TestGenerateDeviceID_Different(t *testing.T) {
	// Different IPs should generate different device IDs
	req1, _ := http.NewRequest("GET", "http://example.com", nil)
	req1.RemoteAddr = "192.168.1.100:12345"
	req1.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	req2, _ := http.NewRequest("GET", "http://example.com", nil)
	req2.RemoteAddr = "192.168.1.200:12345"
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	id1 := GenerateDeviceID(req1)
	id2 := GenerateDeviceID(req2)

	if id1 == id2 {
		t.Errorf("Expected different device IDs for different IPs, got same: %s", id1)
	}
}

func TestGenerateDeviceID_DifferentUserAgent(t *testing.T) {
	// Different User-Agents should generate different device IDs
	req1, _ := http.NewRequest("GET", "http://example.com", nil)
	req1.RemoteAddr = "192.168.1.100:12345"
	req1.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	req2, _ := http.NewRequest("GET", "http://example.com", nil)
	req2.RemoteAddr = "192.168.1.100:12345"
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")

	id1 := GenerateDeviceID(req1)
	id2 := GenerateDeviceID(req2)

	if id1 == id2 {
		t.Errorf("Expected different device IDs for different UAs, got same: %s", id1)
	}
}

func TestGenerateDeviceID_XForwardedFor(t *testing.T) {
	// Should use X-Forwarded-For when present
	req1, _ := http.NewRequest("GET", "http://example.com", nil)
	req1.RemoteAddr = "10.0.0.1:12345" // Internal proxy IP
	req1.Header.Set("X-Forwarded-For", "203.0.113.1")
	req1.Header.Set("User-Agent", "Mozilla/5.0")

	req2, _ := http.NewRequest("GET", "http://example.com", nil)
	req2.RemoteAddr = "203.0.113.1:12345" // Direct IP
	req2.Header.Set("User-Agent", "Mozilla/5.0")

	// Should generate same ID (both use 203.0.113.1)
	id1 := GenerateDeviceID(req1)
	id2 := GenerateDeviceID(req2)

	if id1 != id2 {
		t.Errorf("Expected X-Forwarded-For to be used, got different IDs: %s vs %s", id1, id2)
	}
}

func TestGenerateDeviceID_XForwardedForMultiple(t *testing.T) {
	// Should use first IP in X-Forwarded-For chain
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.2, 10.0.0.3")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	id := GenerateDeviceID(req)

	// Create request with direct IP
	req2, _ := http.NewRequest("GET", "http://example.com", nil)
	req2.RemoteAddr = "203.0.113.1:12345"
	req2.Header.Set("User-Agent", "Mozilla/5.0")

	id2 := GenerateDeviceID(req2)

	// Should match (both use 203.0.113.1)
	if id != id2 {
		t.Errorf("Expected first IP from X-Forwarded-For to be used")
	}
}

func TestGenerateDeviceID_XRealIP(t *testing.T) {
	// Should use X-Real-IP when present
	req1, _ := http.NewRequest("GET", "http://example.com", nil)
	req1.RemoteAddr = "10.0.0.1:12345"
	req1.Header.Set("X-Real-IP", "203.0.113.1")
	req1.Header.Set("User-Agent", "Mozilla/5.0")

	req2, _ := http.NewRequest("GET", "http://example.com", nil)
	req2.RemoteAddr = "203.0.113.1:12345"
	req2.Header.Set("User-Agent", "Mozilla/5.0")

	id1 := GenerateDeviceID(req1)
	id2 := GenerateDeviceID(req2)

	if id1 != id2 {
		t.Errorf("Expected X-Real-IP to be used")
	}
}

func TestGenerateDeviceID_XForwardedForPriority(t *testing.T) {
	// X-Forwarded-For should take priority over X-Real-IP
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	req.Header.Set("X-Real-IP", "203.0.113.2")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	id := GenerateDeviceID(req)

	// Create request with X-Forwarded-For IP
	req2, _ := http.NewRequest("GET", "http://example.com", nil)
	req2.RemoteAddr = "203.0.113.1:12345"
	req2.Header.Set("User-Agent", "Mozilla/5.0")

	id2 := GenerateDeviceID(req2)

	if id != id2 {
		t.Errorf("Expected X-Forwarded-For to take priority")
	}
}

func TestGenerateDeviceID_IPv6(t *testing.T) {
	// Should handle IPv6 addresses
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.RemoteAddr = "[2001:db8::1]:12345"
	req.Header.Set("User-Agent", "Mozilla/5.0")

	id := GenerateDeviceID(req)

	// Should generate valid 64-char hex string
	if len(id) != 64 {
		t.Errorf("Expected 64 character device ID for IPv6, got %d", len(id))
	}
}

func TestGenerateDeviceID_EmptyUserAgent(t *testing.T) {
	// Should handle empty User-Agent
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	// No User-Agent header

	id := GenerateDeviceID(req)

	// Should still generate valid ID
	if len(id) != 64 {
		t.Errorf("Expected valid device ID even without UA, got %d chars", len(id))
	}

	// Two requests from same IP with no UA should match
	req2, _ := http.NewRequest("GET", "http://example.com", nil)
	req2.RemoteAddr = "192.168.1.100:54321"

	id2 := GenerateDeviceID(req2)

	if id != id2 {
		t.Errorf("Expected same ID for same IP with empty UA")
	}
}


func TestGetClientIP_Direct(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	ip := GetClientIP(req)

	if ip != "192.168.1.100" {
		t.Errorf("Expected IP 192.168.1.100, got %s", ip)
	}
}

func TestGetClientIP_XForwardedFor(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")

	ip := GetClientIP(req)

	if ip != "203.0.113.1" {
		t.Errorf("Expected IP from X-Forwarded-For, got %s", ip)
	}
}

func TestGetClientIP_XRealIP(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Real-IP", "203.0.113.1")

	ip := GetClientIP(req)

	if ip != "203.0.113.1" {
		t.Errorf("Expected IP from X-Real-IP, got %s", ip)
	}
}

func TestGetClientIP_IPv6WithBrackets(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.RemoteAddr = "[2001:db8::1]:12345"

	ip := GetClientIP(req)

	if ip != "2001:db8::1" {
		t.Errorf("Expected IPv6 without brackets, got %s", ip)
	}
}

