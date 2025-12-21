package buddy

import (
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/logger"
	"github.com/MunifTanjim/stremthru/internal/peer"
	ti "github.com/MunifTanjim/stremthru/internal/torrent_info"
	ts "github.com/MunifTanjim/stremthru/internal/torrent_stream"
	tss "github.com/MunifTanjim/stremthru/internal/torrent_stream/torrent_stream_syncinfo"
)

// ProwlarrLogger is a callback function for logging peer pulls to the database
// Signature: indexerName, sid, duration, resultsCount, wasSuccessful, errorType, errorMsg
var ProwlarrLogger func(indexerName string, sid string, duration time.Duration, resultsCount int, wasSuccessful bool, errorType, errorMsg string)

var PullPeer, pullLocalOnly = func() (*peer.APIClient, bool) {
	baseUrl := config.PullPeerURL
	if baseUrl == "" {
		baseUrl = config.PeerURL
	}
	localOnly := baseUrl == config.PullPeerURL
	if baseUrl == "" {
		return nil, localOnly
	}
	return peer.NewAPIClient(&peer.APIClientConfig{
		BaseURL: baseUrl,
	}), localOnly
}()

var pullPeerLog = logger.Scoped("peer:pull")

var noTorrentInfo = !config.Feature.HasTorrentInfo()

// supports imdb or anidb
func PullTorrentsByStremId(sid string, originInstanceId string) []string {
	pullPeerLog.Info("PullTorrentsByStremId called",
		"sid", sid,
		"ProwlarrLogger_is_nil", ProwlarrLogger == nil)

	pullPeerLog.Debug("PullTorrentsByStremId called",
		"sid", sid,
		"noTorrentInfo", noTorrentInfo,
		"PullPeer", PullPeer != nil,
		"IsHaltedCheckMagnet", PullPeer != nil && PullPeer.IsHaltedCheckMagnet(),
		"ShouldPull", tss.ShouldPull(sid))

	if noTorrentInfo || PullPeer == nil || PullPeer.IsHaltedCheckMagnet() || !tss.ShouldPull(sid) {
		reason := func() string {
			if noTorrentInfo {
				return "noTorrentInfo"
			}
			if PullPeer == nil {
				return "PullPeer is nil"
			}
			if PullPeer.IsHaltedCheckMagnet() {
				return "CheckMagnet is halted"
			}
			if !tss.ShouldPull(sid) {
				return "ShouldPull returned false (recently pulled/cached)"
			}
			return "unknown"
		}()

		pullPeerLog.Debug("PullTorrentsByStremId early return", "reason", reason)

		// Log cached/skipped searches to database as well
		// This helps track total search volume even when results are cached
		if ProwlarrLogger != nil && reason == "ShouldPull returned false (recently pulled/cached)" {
			// Get torrent count from database for this stream ID
			cleanSId := ts.CleanStremId(sid)
			data, err := ti.ListByStremId(cleanSId, false)
			count := 0
		if err == nil && data != nil {
			count = len(data.Items)
		}

		if count > 0 {
			pullPeerLog.Info("logging cached search", "sid", cleanSId, "count", count)
			// Log cached search with per-indexer breakdown
			if ProwlarrLogger != nil {
				// Group cached results by indexer
				indexerMap := make(map[string]int)
				for i := range data.Items {
					indexerName := data.Items[i].Indexer
					if indexerName == "" {
						indexerName = "Prowlarr (All)" // Fallback
					}
					indexerMap[indexerName]++
				}

				// Log each indexer's cached results
				for indexerName, indexerCount := range indexerMap {
					pullPeerLog.Info("logging cached results per-indexer", "sid", cleanSId, "indexer", indexerName, "count", indexerCount)
					// Log as cached search (0ms response time since it's from cache)
					go ProwlarrLogger(indexerName, cleanSId, 0, indexerCount, true, "", "")
				}
			}
		}
		}

		return nil
	}

	cleanSId := ts.CleanStremId(sid)
	start := time.Now()
	res, err := PullPeer.ListTorrents(&peer.ListTorrentsByStremIdParams{
		SId:              cleanSId,
		LocalOnly:        pullLocalOnly,
		OriginInstanceId: originInstanceId,
	})
	duration := time.Since(start)

	if err != nil {
		if duration > 25*time.Second {
			PullPeer.HaltCheckMagnet()
		}

		pullPeerLog.Error("failed to pull torrents", "error", core.PackError(err), "duration", duration, "sid", cleanSId)

		// Log failed Prowlarr search to database via callback
		if ProwlarrLogger != nil {
			go ProwlarrLogger("Prowlarr (All)", cleanSId, duration, 0, false, "peer_error", err.Error())
		}

		return nil
	}

	count := len(res.Data.Items)
	pullPeerLog.Info("pulled torrents", "duration", duration, "sid", cleanSId, "count", count)

	// DEBUG: Log that we're about to process results
	pullPeerLog.Info("DEBUG: About to log results", "count", count, "ProwlarrLogger_is_nil", ProwlarrLogger == nil)

	// Log search results to database via callback
	// Log each result's indexer separately for per-indexer tracking
	if count > 0 && ProwlarrLogger != nil {
		pullPeerLog.Info("DEBUG: Processing results for logging", "sid", cleanSId, "total_results", count, "duration_ms", duration.Milliseconds())

		// Group by indexer for logging
		indexerMap := make(map[string]int)
		for i := range res.Data.Items {
			indexerName := res.Data.Items[i].Indexer
			pullPeerLog.Debug("DEBUG: Processing result item", "index", i, "indexer", indexerName, "title", res.Data.Items[i].TorrentTitle)
			if indexerName == "" {
				pullPeerLog.Warn("DEBUG: Result has empty indexer field", "index", i, "title", res.Data.Items[i].TorrentTitle)
				indexerName = "Prowlarr (All)" // Fallback for results without indexer info
			}
			indexerMap[indexerName]++
		}

		pullPeerLog.Info("DEBUG: Indexer map created", "indexer_count", len(indexerMap))

		// Log each indexer separately
		for indexerName, indexerCount := range indexerMap {
			pullPeerLog.Info("DEBUG: About to call ProwlarrLogger", "indexer", indexerName, "count", indexerCount)
			go ProwlarrLogger(indexerName, cleanSId, duration, indexerCount, true, "", "")
			pullPeerLog.Info("DEBUG: ProwlarrLogger called for", "indexer", indexerName)
		}
	} else if count == 0 && ProwlarrLogger != nil {
		// Log empty results
		pullPeerLog.Info("DEBUG: No results returned", "sid", cleanSId)
		go ProwlarrLogger("Prowlarr (All)", cleanSId, duration, 0, true, "", "")
	}

	hashes := make([]string, count)
	items := make([]ti.TorrentInfoInsertData, count)
	for i := range res.Data.Items {
		data := &res.Data.Items[i]
		hashes[i] = data.Hash
		items[i] = ti.TorrentInfoInsertData{
			Hash:         data.Hash,
			TorrentTitle: data.TorrentTitle,
			Size:         data.Size,
			Indexer:      data.Indexer,
			Source:       ti.TorrentInfoSource(data.Source),
			Category:     ti.TorrentInfoCategory(data.Category),
			Files:        data.Files,
			Seeders:      data.Seeders,
			Leechers:     data.Leechers,
			Private:      data.Private,
		}
	}
	ti.Upsert(items, "", false)
	go tss.MarkPulled(cleanSId)
	return hashes
}

func ListTorrentsByStremId(sid string, localOnly bool, originInstanceId string, noMissingSize bool) (*ti.ListTorrentsData, error) {
	if originInstanceId == config.InstanceId && !pullLocalOnly {
		pullPeerLog.Info("loop detected for list torrents, self-correcting...")
		pullLocalOnly = true
	}

	if !localOnly {
		PullTorrentsByStremId(sid, originInstanceId)
	}

	data, err := ti.ListByStremId(sid, noMissingSize)
	if err != nil {
		return nil, err
	}
	return data, nil
}

