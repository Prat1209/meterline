-- +goose Up
CREATE TABLE accounts (
    id           BIGSERIAL PRIMARY KEY,
    name         TEXT        NOT NULL,
    api_key_hash BYTEA       NOT NULL UNIQUE, -- sha256 of the API key; the raw key is never stored
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE idempotency_keys (
    account_id      BIGINT      NOT NULL REFERENCES accounts (id),
    key             TEXT        NOT NULL,
    request_hash    BYTEA       NOT NULL, -- sha256 of method, path and canonical body
    response_status INT         NOT NULL,
    response_body   BYTEA       NOT NULL, -- exact bytes sent, so replays are byte-identical
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, key)
);

CREATE TABLE usage_events (
    id              BIGSERIAL PRIMARY KEY,
    account_id      BIGINT      NOT NULL REFERENCES accounts (id),
    idempotency_key TEXT        NOT NULL,
    model           TEXT        NOT NULL,
    input_tokens    BIGINT      NOT NULL CHECK (input_tokens >= 0),
    output_tokens   BIGINT      NOT NULL CHECK (output_tokens >= 0),
    occurred_at     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (account_id, idempotency_key) -- second line of defence against duplicates
);

CREATE INDEX usage_events_account_occurred_at ON usage_events (account_id, occurred_at);

-- +goose Down
DROP TABLE usage_events;
DROP TABLE idempotency_keys;
DROP TABLE accounts;
