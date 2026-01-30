package stremio_userdata

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/MunifTanjim/stremthru/internal/cache"
	torznab_client "github.com/MunifTanjim/stremthru/internal/torznab/client"
	"github.com/MunifTanjim/stremthru/internal/torznab/jackett"
)

type IndexerName string

const (
	IndexerNameGeneric   IndexerName = "generic"
	IndexerNameJackett   IndexerName = "jackett"
	IndexerNameProwlarr  IndexerName = "prowlarr"
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

// prowlarrTorznabClient wraps torznab_client.Client to implement the Indexer interface
type prowlarrTorznabClient struct {
	*torznab_client.Client
	id string
}

func (ptc *prowlarrTorznabClient) GetId() string {
	return "prowlarr/" + ptc.id
}

func (ptc *prowlarrTorznabClient) Search(query *torznab_client.Query) ([]torznab_client.Torz, error) {
	// Prowlarr uses the same Torznab protocol as Jackett
	// We'll use jackett's search response type since Prowlarr implements the same protocol
	params := &jackett.Ctx{}
	q := query.Values()
	params.Query = &q

	var resp torznab_client.Response[jackett.SearchResponse]
	_, err := ptc.Client.Request("GET", "/api", params, &resp)
	if err != nil {
		return nil, err
	}

	items := resp.Data.Channel.Items
	result := make([]torznab_client.Torz, 0, len(items))
	for i := range items {
		item := &items[i]
		result = append(result, *item.ToTorz())
	}
	return result, nil
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
			// Prowlarr acts as a Torznab indexer aggregator
			// Use the base URL + /api/v2.0/indexers/all/results/torznab as the endpoint
			torznabURL := baseURL + "/api/v2.0/indexers/all/results/torznab"

			tc := torznab_client.NewClient(&torznab_client.ClientConfig{
				BaseURL: torznabURL,
				APIKey:  apiKey,
			})

			// Wrap the torznab client to implement the Indexer interface
			client := &prowlarrTorznabClient{
				Client: tc,
				id:     "all",
			}
			indexers = append(indexers, client)

		default:
			return indexers, errors.New("unsupported indexer: " + string(indexer.Name))
		}
	}
	return indexers, nil
}


