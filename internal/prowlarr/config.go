package prowlarr

import (
	"os"
)

var (
	Enabled bool
	URL     string
	APIKey  string
)

func init() {
	Enabled = os.Getenv("PROWLARR_ENABLED") == "true"
	URL = os.Getenv("PROWLARR_URL")
	APIKey = os.Getenv("PROWLARR_API_KEY")

	if URL == "" {
		URL = "http://localhost:9696"
	}
}

// IsConfigured returns true if Prowlarr is properly configured
func IsConfigured() bool {
	return Enabled && URL != "" && APIKey != ""
}

