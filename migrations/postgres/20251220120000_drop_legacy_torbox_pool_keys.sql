-- +goose Up
-- +goose StatementBegin

-- Drop the legacy torbox_pool_keys table
-- This table has been superseded by torbox_pool which has better
-- tracking for user slots and concurrent streams
DROP TABLE IF EXISTS "public"."torbox_pool_keys";

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- This migration is IRREVERSIBLE
-- The legacy table has been replaced by the improved torbox_pool table
-- that includes proper slot management and concurrency tracking.
-- Do not attempt to rollback this migration in production.

-- +goose StatementEnd

