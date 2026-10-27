-- +goose Up
CREATE TABLE devices (
    id                      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    phone                   TEXT        NOT NULL UNIQUE,
    name                    TEXT        NOT NULL DEFAULT '',
    timezone_offset_minutes SMALLINT    NOT NULL DEFAULT 420,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at            TIMESTAMPTZ
);

COMMENT ON COLUMN devices.phone IS 'Nomor terminal dari header JT808 (BCD, 12 digit)';
COMMENT ON COLUMN devices.timezone_offset_minutes IS 'Selisih zona waktu jam perangkat terhadap UTC, dalam menit (UTC+7 = 420)';

-- +goose Down
DROP TABLE devices;