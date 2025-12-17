package device

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

// GenerateDeviceID creates consistent device ID from IP + User-Agent
func GenerateDeviceID(r *http.Request) string {
	ip := GetClientIP(r)
	ua := r.Header.Get("User-Agent")

	hash := sha256.Sum256([]byte(ip + "|" + ua))
	return hex.EncodeToString(hash[:])
}

// GetClientIP extracts the client IP from the request, checking proxy headers
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (proxy/load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take first IP if multiple
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header (nginx/other proxies)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fallback to RemoteAddr (remove port)
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	// Remove brackets from IPv6 addresses
	ip = strings.Trim(ip, "[]")

	return ip
}

