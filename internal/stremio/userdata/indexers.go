package stremio_userdata

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	"github.com/MunifTanjim/stremthru/internal/prowlarr"
	torznab_client "github.com/MunifTanjim/stremthru/internal/torznab/client"
	"github.com/MunifTanjim/stremthru/internal/torznab/jackett"
)

type IndexerName string

const (
	IndexerNameGeneric  IndexerName = "generic"
	IndexerNameJackett  IndexerName = "jackett"
	IndexerNameProwlarr IndexerName = "prowlarr"
)

type Indexer struct {
	Name   IndexerName `json:"n"`
	URL    string      `json:"u"`
	APIKey string      `json:"ak,omitempty"`
}

// Global DB instance for indexer logging (set by main.go)
var IndexerDB *sql.DB

// InitializeIndexerDB sets the database connection for Prowlarr logging
func InitializeIndexerDB(db *sql.DB) {
	IndexerDB = db
}

// type rawIndexer Indexer
//
// func (i Indexer) MarshalJSON() ([]byte, error) {
// 	i.Compress()
// 	return json.Marshal(rawIndexer(i))
// }
//
// func (i *Indexer) UnmarshalJSON(data []byte) error {
// 	ri := rawIndexer{}
// 	err := json.Unmarshal(data, &ri)
// 	if err != nil {
// 		return err
// 	}
// 	*i = Indexer(ri)
// 	i.Decompress()
// 	return nil
// }

func (i *Indexer) Decompress() {
	switch i.Name {
	case IndexerNameJackett:
		i.URL = jackett.TorznabURL(i.URL).Decode()
	}
}

func (i *Indexer) Compress() {
	switch i.Name {
	case IndexerNameJackett:
		i.URL = jackett.TorznabURL(i.URL).Encode()
	}
}

func (i Indexer) Validate() (string, error) {
	if i.Name == "" {
		return "name", fmt.Errorf("indexer name is required")
	}
	if i.URL == "" {
		return "url", fmt.Errorf("indexer url is required")
	}
	switch i.Name {
	case IndexerNameJackett:
		if err := jackett.TorznabURL(i.URL).Parse(); err != nil {
			return "url", fmt.Errorf("indexer url is invalid")
		}
	}
	return "", nil
}

type UserDataIndexers struct {
	Indexers []Indexer `json:"indexers"`
}

func (ud UserDataIndexers) HasRequiredValues() bool {
	for i := range ud.Indexers {
		indexer := &ud.Indexers[i]
		if _, err := indexer.Validate(); err != nil {
			return false
		}
	}
	return true
}

func (ud UserDataIndexers) StripSecrets() UserDataIndexers {
	ud.Indexers = slices.Clone(ud.Indexers)
	for i := range ud.Indexers {
		s := &ud.Indexers[i]
		s.APIKey = ""
	}
	return ud
}

var jackettCache = cache.NewCache[*jackett.Client](&cache.CacheConfig{
	Lifetime: 6 * time.Hour,
	Name:     "stremio:userdata:indexers:jackett",
})

// prowlarrNativeClient uses Prowlarr's native /api/v1/search API
type prowlarrNativeClient struct {
	id             string
	prowlarrClient *prowlarr.Client
}

func (pnc *prowlarrNativeClient) GetId() string {
	return "prowlarr/" + pnc.id
}

// createProwlarrFakeCaps creates a Caps struct with all search functions enabled
func createProwlarrFakeCaps() *torznab_client.Caps {
	caps := &torznab_client.Caps{}
	caps.Server.Title = "Prowlarr"

	// Enable all search types
	caps.Searching.Search.Available = true
	caps.Searching.Search.SupportedParams = "q"

	caps.Searching.TVSearch.Available = true
	caps.Searching.TVSearch.SupportedParams = "q,season,ep,imdbid,tvdbid"

	caps.Searching.MovieSearch.Available = true
	caps.Searching.MovieSearch.SupportedParams = "q,imdbid,tmdbid"

	return caps
}

func (pnc *prowlarrNativeClient) NewSearchQuery(fn func(caps torznab_client.Caps) torznab_client.Function) (*torznab_client.Query, error) {
	// Create fake caps without calling GetCaps (which would fail with 401)
	fakeCaps := createProwlarrFakeCaps()

	// Use the new constructor to create a Query with our fake caps
	query := torznab_client.NewQueryWithCaps(fakeCaps, fn)
	return query, nil
}

func (pnc *prowlarrNativeClient) Search(query *torznab_client.Query) ([]torznab_client.Torz, error) {
	values := query.Values()

	// Extract search parameters
	var searchQuery string
	var searchType string

	t := values.Get("t")
	switch t {
	case "movie":
		searchType = "movie"
	case "tvsearch":
		searchType = "tvsearch"
	default:
		searchType = "search"
	}

	// Get search query from various parameters
	if imdbId := values.Get("imdbid"); imdbId != "" {
		searchQuery = imdbId
	} else if q := values.Get("q"); q != "" {
		searchQuery = q
	}

	if searchQuery == "" {
		return []torznab_client.Torz{}, nil
	}

	// Call Prowlarr native API
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := pnc.prowlarrClient.Search(ctx, searchQuery, searchType)
	if err != nil {
		return nil, fmt.Errorf("prowlarr native search failed: %w", err)
	}

	// Convert Prowlarr results to Torz format
	torrents := make([]torznab_client.Torz, 0, len(results))
	for i := range results {
		result := &results[i]

		// Create magnet link from infohash
		magnetLink := ""
		if result.InfoHash != "" {
			magnetLink = "magnet:?xt=urn:btih:" + strings.ToLower(result.InfoHash)
		}

		torz := torznab_client.Torz{
			Indexer:    result.Indexer,
			Hash:       strings.ToLower(result.InfoHash),
			Title:      result.Title,
			Size:       result.Size,
			Seeders:    result.Seeders,
			Leechers:   result.Leechers,
			MagnetLink: magnetLink,
			SourceLink: result.DownloadURL,
		}
		torrents = append(torrents, torz)
	}

	return torrents, nil
}

func (ud *UserDataIndexers) Compress() {
	for i := range ud.Indexers {
		indexer := &ud.Indexers[i]
		indexer.Compress()
	}
}

func (ud *UserDataIndexers) Decompress() {
	for i := range ud.Indexers {
		indexer := &ud.Indexers[i]
		indexer.Decompress()
	}
}

func (ud *UserDataIndexers) Prepare() ([]torznab_client.Indexer, error) {
	indexers := make([]torznab_client.Indexer, 0, len(ud.Indexers))
	for i := range ud.Indexers {
		indexer := &ud.Indexers[i]

		baseURL := indexer.URL
		apiKey := indexer.APIKey

		switch indexer.Name {
		case IndexerNameJackett:
			u := jackett.TorznabURL(baseURL)
			if err := u.Parse(); err != nil {
				return indexers, err
			}

			key := u.BaseURL + ":" + apiKey
			var client *jackett.Client
			if !jackettCache.Get(key, &client) {
				client = jackett.NewClient(&jackett.ClientConfig{
					BaseURL: u.BaseURL,
					APIKey:  apiKey,
				})
				err := jackettCache.Add(key, client)
				if err != nil {
					return indexers, err
				}
			}
			c := client.GetTorznabClient(u.IndexerId)
			indexers = append(indexers, c)

		case IndexerNameProwlarr:
			// Use Prowlarr's native API instead of the broken Torznab v2 endpoint
			prowlarrClient := prowlarr.NewClient(&prowlarr.ClientConfig{
				BaseURL: baseURL,
				APIKey:  apiKey,
			})

			client := &prowlarrNativeClient{
				id:             "all",
				prowlarrClient: prowlarrClient,
			}
			indexers = append(indexers, client)

		default:
			return indexers, errors.New("unsupported indexer: " + string(indexer.Name))
		}
	}
	return indexers, nil
}
