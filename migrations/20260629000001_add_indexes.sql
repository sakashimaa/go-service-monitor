-- +goose Up
-- +goose StatementBegin
CREATE INDEX idx_site_checks_site_id_created_at
ON site_checks(site_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_site_checks_site_id_created_at;
-- +goose StatementEnd