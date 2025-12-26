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
	// Log using the standard chillproxy logging pattern
	log.Info("chillstreams config check", "enableAuth", config.EnableChillstreamsAuth, "apiURL", config.ChillstreamsAPIURL, "hasAPIKey", config.ChillstreamsAPIKey != "")

	if !config.EnableChillstreamsAuth {
		log.Debug("chillstreams auth disabled, skipping")
		return nil
	}

	client := getChillstreamsClient()
	if client == nil {
		log.Debug("chillstreams client not initialized", "apiKeyEmpty", config.ChillstreamsAPIKey == "", "apiUrlEmpty", config.ChillstreamsAPIURL == "")
		return nil // Chillstreams not configured, skip
	}

	log.Info("chillstreams client ready")
	deviceID := device.GenerateDeviceID(r)
	log.Debug("device id generated", "deviceId", deviceID)

	storeCount := 0
	for i := range ud.stores {
		s := &ud.stores[i]
		storeCount++

		if s.ChillstreamsAuth == "" {
			log.Debug("store skipped - no chillstreams auth", "store", s.Store.GetName(), "index", i)
			continue // No Chillstreams auth for this store
		}

		log.Info("requesting chillstreams pool key", "userId", s.ChillstreamsAuth, "store", s.Store.GetName())

		// Fetch pool key from Chillstreams
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		resp, err := client.GetPoolKey(ctx, chillstreams.GetPoolKeyRequest{
			UserID:   s.ChillstreamsAuth,
			DeviceID: deviceID,
			Action:   "init",
		})

		if err != nil {
			log.Error("failed to get chillstreams pool key", "error", err, "userId", s.ChillstreamsAuth)
			return fmt.Errorf("chillstreams authentication failed: %w", err)
		}

		if !resp.Allowed {
			log.Warn("chillstreams user not allowed", "userId", s.ChillstreamsAuth, "message", resp.Message)
			return fmt.Errorf("authentication failed: %s", resp.Message)
		}

		// Store pool key ID for usage logging
		s.PoolKeyID = resp.PoolKeyID

		// Inject pool key into store client
		if resp.PoolKey != "" {
			log.Debug("pool key received from chillstreams", "poolKey", resp.PoolKey, "poolKeyLength", len(resp.PoolKey), "poolKeyId", resp.PoolKeyID)

			switch client := s.Store.(type) {
			case *torbox.StoreClient:
				// Verify the pool key before injection
				if len(resp.PoolKey) != 36 {
					log.Error("invalid pool key format", "poolKey", resp.PoolKey, "expectedLength", 36, "actualLength", len(resp.PoolKey))
					return fmt.Errorf("invalid pool key format received from chillstreams")
				}

				// CRITICAL: Set BOTH client API key AND AuthToken
				// client.SetAPIKey affects the TorBox client's internal state
				// s.AuthToken is used in CheckMagnet params
				client.SetAPIKey(resp.PoolKey)
				s.AuthToken = resp.PoolKey // ✅ This is used by CheckMagnet

				// Verify the key was set correctly
				actualKey := client.GetAPIKey()
				log.Info("💛 torpool pool key injected", "userId", s.ChillstreamsAuth, "poolKeyId", resp.PoolKeyID, "deviceCount", resp.DeviceCount, "store", s.Store.GetName(), "keySet", actualKey == resp.PoolKey, "keyLength", len(actualKey), "authTokenSet", s.AuthToken == resp.PoolKey)

				if actualKey != resp.PoolKey {
					log.Error("API key mismatch after setting", "expected", resp.PoolKey, "actual", actualKey)
					return fmt.Errorf("failed to set TorBox API key correctly")
				}

				if s.AuthToken != resp.PoolKey {
					log.Error("AuthToken mismatch after setting", "expected", resp.PoolKey, "actual", s.AuthToken)
					return fmt.Errorf("failed to set AuthToken correctly")
				}
			default:
				log.Debug("chillstreams auth not supported for this store type", "store", s.Store.GetName())
			}
		} else {
			log.Error("empty pool key received from chillstreams", "userId", s.ChillstreamsAuth, "poolKeyId", resp.PoolKeyID)
			return fmt.Errorf("empty pool key received")
		}
	}

	log.Debug("chillstreams initialization complete", "totalStores", storeCount)


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

