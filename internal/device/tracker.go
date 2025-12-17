package device

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

// GenerateDeviceID creates consistent device ID from IP + User-Agent
func GenerateDeviceID(r *http.Request) string {
	ip := getClientIP(r)
	ua := r.Header.Get("User-Agent")

	hash := sha256.Sum256([]byte(ip + "|" + ua))
	return hex.EncodeToString(hash[:])
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

