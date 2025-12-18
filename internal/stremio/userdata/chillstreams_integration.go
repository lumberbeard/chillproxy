package stremio_userdata

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/MunifTanjim/stremthru/internal/chillstreams"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/device"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/store/torbox"
)

var chillstreamsClient *chillstreams.Client
var chillstreamsClientInit bool

func getChillstreamsClient() *chillstreams.Client {
	if !chillstreamsClientInit {
		if config.ChillstreamsAPIKey != "" {
			chillstreamsClient = chillstreams.NewClient(config.ChillstreamsAPIURL, config.ChillstreamsAPIKey)
		}
		chillstreamsClientInit = true
	}
	return chillstreamsClient
}

// InitializeStoresWithChillstreams fetches pool keys from Chillstreams and injects them into stores
func (ud *UserDataStores) InitializeStoresWithChillstreams(r *http.Request, log *logger.Logger) error {
	// Explicit logging that will definitely show up
	fmt.Printf("\n=== CHILLSTREAMS INTEGRATION DEBUG START ===\n")
	fmt.Printf("EnableChillstreamsAuth: %v\n", config.EnableChillstreamsAuth)
	fmt.Printf("ChillstreamsAPIURL: %s\n", config.ChillstreamsAPIURL)
	fmt.Printf("ChillstreamsAPIKey: %s\n", config.ChillstreamsAPIKey)
	fmt.Printf("=== CHILLSTREAMS INTEGRATION DEBUG END ===\n\n")

	log.Info("🔵 [CHILLSTREAMS] Checking integration enabled", "enabled", config.EnableChillstreamsAuth, "apiURL", config.ChillstreamsAPIURL)

	if !config.EnableChillstreamsAuth {
		log.Info("🔵 [CHILLSTREAMS] Auth disabled, skipping")
		return nil
	}

	client := getChillstreamsClient()
	if client == nil {
		log.Warn("🔵 [CHILLSTREAMS] Client is nil (not configured properly)")
		return nil // Chillstreams not configured, skip
	}

	log.Info("🔵 [CHILLSTREAMS] Client initialized successfully")
	deviceID := device.GenerateDeviceID(r)
	log.Debug("🔵 [CHILLSTREAMS] Generated device id", "deviceId", deviceID)

	for i := range ud.stores {
		s := &ud.stores[i]
		log.Debug("🔵 [CHILLSTREAMS] Checking store", "store", s.Store.GetName(), "hasAuth", s.ChillstreamsAuth != "")

		if s.ChillstreamsAuth == "" {
			log.Debug("🔵 [CHILLSTREAMS] No chillstreams auth for this store", "store", s.Store.GetName())
			continue // No Chillstreams auth for this store
		}

		log.Info("🔵 [CHILLSTREAMS] Requesting pool key", "userId", s.ChillstreamsAuth, "store", s.Store.GetName())

		// Fetch pool key from Chillstreams
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		resp, err := client.GetPoolKey(ctx, chillstreams.GetPoolKeyRequest{
			UserID:   s.ChillstreamsAuth,
			DeviceID: deviceID,
			Action:   "init",
		})

		if err != nil {
			log.Error("🔵 [CHILLSTREAMS] Failed to get pool key", "error", err, "userId", s.ChillstreamsAuth)
			return fmt.Errorf("chillstreams authentication failed: %w", err)
		}

		if !resp.Allowed {
			log.Warn("🔵 [CHILLSTREAMS] User not allowed", "userId", s.ChillstreamsAuth, "message", resp.Message)
			return fmt.Errorf("authentication failed: %s", resp.Message)
		}

		// Store pool key ID for usage logging
		s.PoolKeyID = resp.PoolKeyID

		// Inject pool key into store client
		if resp.PoolKey != "" {
			switch client := s.Store.(type) {
			case *torbox.StoreClient:
				client.SetAPIKey(resp.PoolKey)
				s.AuthToken = resp.PoolKey // Update auth token for other methods
				log.Info("💛 TORPOOL 💛 Injected pool key for TorBox", "userId", s.ChillstreamsAuth, "poolKeyId", resp.PoolKeyID, "deviceCount", resp.DeviceCount)
			default:
				log.Warn("🔵 [CHILLSTREAMS] Auth not supported for this store type", "store", s.Store.GetName())
			}
		}

		log.Info("💛 TORPOOL 💛 Chillstreams initialization complete", "userId", s.ChillstreamsAuth, "store", s.Store.GetName())

	}

	return nil
}

// LogChillstreamsUsage logs usage to Chillstreams for stores using Chillstreams auth
func (ud *UserDataStores) LogChillstreamsUsage(hash string, cached bool, bytes int64) {
	if !config.EnableChillstreamsAuth {
		return
	}

	client := getChillstreamsClient()
	if client == nil {
		return
	}

	for i := range ud.stores {
		s := &ud.stores[i]
		if s.ChillstreamsAuth == "" || s.PoolKeyID == "" {
			continue
		}

		// Log usage asynchronously (fire and forget)
		go func(userID, poolKeyID, hash string, cached bool, bytes int64) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := client.LogUsage(ctx, chillstreams.LogUsageRequest{
				UserID:    userID,
				PoolKeyID: poolKeyID,
				Action:    "stream-served",
				Hash:      hash,
				Cached:    cached,
				Bytes:     bytes,
			})

			if err != nil {
				// Log error but don't fail the request
				// (usage logging is non-critical)
			}
		}(s.ChillstreamsAuth, s.PoolKeyID, hash, cached, bytes)
	}
}

