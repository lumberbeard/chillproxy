package prowlarr

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// LogSearchRequest logs a Prowlarr search to the database for performance tracking
type SearchLogParams struct {
	IndexerName   string        // e.g., "EZTV", "The Pirate Bay", etc.
	SearchQuery   string        // Search query text
	SearchType    string        // "movie", "series", "search"
	IMDBId        string        // Optional IMDB ID
	TMDBId        string        // Optional TMDB ID
	ResponseTime  time.Duration // How long the search took
	HTTPStatus    int           // HTTP status code
	ResultsCount  int           // Number of results returned
	WasSuccessful bool          // Whether the search succeeded
	ErrorType     string        // "timeout", "http_error", "parse_error", "no_results"
	ErrorMessage  string        // Error message if failed
	UserID        string        // Optional user UUID
}

// LogSearchToDatabase logs a Prowlarr search to the PostgreSQL database (Chillstreams)
func LogSearchToDatabase(ctx context.Context, db *sql.DB, params SearchLogParams) error {
	// Find indexer ID by name in Chillstreams PostgreSQL database (case-insensitive)
	var indexerID *uuid.UUID
	err := db.QueryRowContext(ctx,
		`SELECT id FROM prowlarr_indexers WHERE LOWER(indexer_name) = LOWER($1)`,
		params.IndexerName,
	).Scan(&indexerID)

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to find indexer: %w", err)
	}

	// Insert search log into PostgreSQL
	_, err = db.ExecContext(ctx, `
		INSERT INTO prowlarr_search_logs
		(indexer_id, search_query, search_type, imdb_id, tmdb_id,
		 response_time, http_status, results_count, was_successful,
		 error_type, error_message, user_uuid)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`,
		indexerID,
		params.SearchQuery,
		params.SearchType,
		nilIfEmpty(params.IMDBId),
		nilIfEmpty(params.TMDBId),
		int(params.ResponseTime.Milliseconds()),
		params.HTTPStatus,
		params.ResultsCount,
		params.WasSuccessful,
		nilIfEmpty(params.ErrorType),
		nilIfEmpty(params.ErrorMessage),
		nilIfEmpty(params.UserID),
	)

	if err != nil {
		return fmt.Errorf("failed to insert search log: %w", err)
	}

	// Update indexer metrics in PostgreSQL
	if indexerID != nil {
		_, err = db.ExecContext(ctx, `
			UPDATE prowlarr_indexers
			SET
				total_requests_24h = total_requests_24h + 1,
				successful_requests_24h = CASE WHEN $1 THEN successful_requests_24h + 1 ELSE successful_requests_24h END,
				failed_requests_24h = CASE WHEN NOT $1 THEN failed_requests_24h + 1 ELSE failed_requests_24h END,
				timeout_count_24h = CASE WHEN $2 = 'timeout' THEN timeout_count_24h + 1 ELSE timeout_count_24h END,
				last_check_at = CURRENT_TIMESTAMP,
				last_success_at = CASE WHEN $1 THEN CURRENT_TIMESTAMP ELSE last_success_at END,
				last_failure_at = CASE WHEN NOT $1 THEN CURRENT_TIMESTAMP ELSE last_failure_at END,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $3
		`,
			params.WasSuccessful,
			params.ErrorType,
			indexerID,
		)

		if err != nil {
			return fmt.Errorf("failed to update indexer metrics: %w", err)
		}
	}

	return nil
}

// nilIfEmpty returns nil if the string is empty, otherwise returns the string
func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

