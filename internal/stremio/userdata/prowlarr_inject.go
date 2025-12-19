package stremio_userdata

import (
	"github.com/MunifTanjim/stremthru/internal/prowlarr"
)

// InjectProwlarrIndexer automatically injects Prowlarr as an indexer if configured
func (ud *UserDataIndexers) InjectProwlarrIndexer() {
	if !prowlarr.IsConfigured() {
		return
	}

	// Check if Prowlarr is already in the list
	for _, idx := range ud.Indexers {
		if idx.Name == IndexerNameProwlarr {
			return // Already configured
		}
	}

	// Add Prowlarr indexer
	ud.Indexers = append(ud.Indexers, Indexer{
		Name:   IndexerNameProwlarr,
		URL:    prowlarr.URL,
		APIKey: prowlarr.APIKey,
	})
}

