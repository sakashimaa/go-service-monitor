-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS sites (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    response_code INT,
    last_check_time TIMESTAMPTZ,
    response_time BIGINT, -- в миллисекундах
    error TEXT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sites;
-- +goose StatementEnd