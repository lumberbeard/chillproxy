package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/MunifTanjim/stremthru/internal/buddy"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/db"
	"github.com/MunifTanjim/stremthru/internal/endpoint"
	"github.com/MunifTanjim/stremthru/internal/posthog"
	"github.com/MunifTanjim/stremthru/internal/prowlarr"
	"github.com/MunifTanjim/stremthru/internal/shared"
	stremio_userdata "github.com/MunifTanjim/stremthru/internal/stremio/userdata"
	"github.com/MunifTanjim/stremthru/internal/worker"
	"github.com/MunifTanjim/stremthru/store"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	config.PrintConfig(&config.AppState{
		StoreNames: []string{
			string(store.StoreNameAlldebrid),
			string(store.StoreNameDebridLink),
			string(store.StoreNameEasyDebrid),
			string(store.StoreNameOffcloud),
			string(store.StoreNamePikPak),
			string(store.StoreNamePremiumize),
			string(store.StoreNameRealDebrid),
			string(store.StoreNameTorBox),
		},
	})

	posthog.Init()
	defer posthog.Close()

	database := db.Open()
	defer db.Close()
	db.Ping()
	RunSchemaMigration(database.URI, database)

	// Initialize PostgreSQL connection for Chillstreams logging (Prowlarr searches)
	var loggingDB *sql.DB
	chillstreamsURI := os.Getenv("CHILLSTREAMS_DATABASE_URL")
	log.Printf("[PROWLARR] CHILLSTREAMS_DATABASE_URL: %s\n", chillstreamsURI)

	if chillstreamsURI != "" {
		var err error
		log.Println("[PROWLARR] Attempting to connect to Chillstreams PostgreSQL database...")
		loggingDB, err = sql.Open("pgx", chillstreamsURI)
		if err != nil {
			log.Printf("[PROWLARR] ❌ Failed to open PostgreSQL connection: %v\n", err)
		} else {
			log.Println("[PROWLARR] ✅ PostgreSQL driver opened")
			defer loggingDB.Close()
			// Test the connection
			if err := loggingDB.Ping(); err != nil {
				log.Printf("[PROWLARR] ❌ PostgreSQL ping failed: %v\n", err)
				loggingDB = nil
			} else {
				log.Println("[PROWLARR] ✅ PostgreSQL connection successful - Prowlarr logging ENABLED")
			}
		}
	} else {
		log.Println("[PROWLARR] ⚠️  CHILLSTREAMS_DATABASE_URL not set - Prowlarr logging DISABLED")
	}

	// Initialize Prowlarr indexer database for logging
	if loggingDB != nil {
		stremio_userdata.InitializeIndexerDB(loggingDB)
		log.Println("[PROWLARR] IndexerDB initialized for logging")

		log.Println("[PROWLARR] About to set up Prowlarr logger callback...")

		// Set up Prowlarr logger callback for buddy layer
		// Now receives indexer name as first parameter for per-indexer tracking
		buddy.ProwlarrLogger = func(indexerName string, sid string, duration time.Duration, resultsCount int, wasSuccessful bool, errorType, errorMsg string) {
			log.Printf("[PROWLARR] ⚡ Callback invoked for indexer=%s, sid=%s, results=%d, duration=%dms\n", indexerName, sid, resultsCount, duration.Milliseconds())

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			httpStatus := 200
			if !wasSuccessful {
				httpStatus = 500
			}

			err := prowlarr.LogSearchToDatabase(ctx, loggingDB, prowlarr.SearchLogParams{
				IndexerName:   indexerName, // Use the actual indexer name passed in
				SearchQuery:   sid,
				SearchType:    "torrent",
				ResponseTime:  duration,
				HTTPStatus:    httpStatus,
				ResultsCount:  resultsCount,
				WasSuccessful: wasSuccessful,
				ErrorType:     errorType,
				ErrorMessage:  errorMsg,
			})

			if err != nil {
				log.Printf("[PROWLARR] ❌ Failed to log search: %v\n", err)
			} else {
				log.Printf("[PROWLARR] ✅ Logged search for %s - %s: %d results in %dms\n", indexerName, sid, resultsCount, duration.Milliseconds())
			}
		}
		log.Println("[PROWLARR] ✅ Callback logger set up for peer pulls")
	} else {
		log.Println("[PROWLARR] ⚠️  IndexerDB is nil - logging will not work")
	}

	stopWorkers := worker.InitWorkers()
	defer stopWorkers()

	mux := http.NewServeMux()

	endpoint.AddRootEndpoint(mux)
	endpoint.AddDashEndpoint(mux)
	endpoint.AddAuthEndpoints(mux)
	endpoint.AddHealthEndpoints(mux)
	endpoint.AddMetaEndpoints(mux)
	endpoint.AddProxyEndpoints(mux)
	endpoint.AddStoreEndpoints(mux)
	endpoint.AddStremioEndpoints(mux)
	endpoint.AddTorrentEndpoints(mux)
	endpoint.AddTorznabEndpoints(mux)
	endpoint.AddExperimentEndpoints(mux)

	handler := shared.RootServerContext(mux)

	addr := ":" + config.Port
	if config.Environment == config.EnvDev {
		addr = "localhost" + addr
	}
	server := &http.Server{Addr: addr, Handler: handler}

	if len(config.ProxyAuthPassword) == 0 {
		server.SetKeepAlivesEnabled(false)
	}

	log.Println("stremthru listening on " + addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to start stremthru: %v", err)
	}
}
