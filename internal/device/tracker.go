package device

import (
	"net/http"
	"strings"
)

// GenerateDeviceID creates consistent device ID
// SIMPLIFIED: Using constant "default-device" since IP-based detection is unreliable
// IP addresses change between requests (IPv4/IPv6 switching, proxy headers, load balancing)
// This ensures one device assignment per user, preventing race conditions
// TODO: Frontend-based device fingerprinting (localStorage-based UUID from Stremio)
func GenerateDeviceID(r *http.Request) string {
	// Return constant device ID to ensure stability
	// This effectively gives each user ONE device assignment for the pool
	return "default-device"
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For, X-Real-IP headers
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Extract IP from RemoteAddr (remove port)
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

