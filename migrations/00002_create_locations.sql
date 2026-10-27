-- +goose Up
CREATE TABLE locations (
    device_id     BIGINT           NOT NULL REFERENCES devices (id),
    device_time   TIMESTAMPTZ      NOT NULL,
    received_at   TIMESTAMPTZ      NOT NULL DEFAULT now(),
    latitude      DOUBLE PRECISION NOT NULL CHECK (latitude BETWEEN -90 AND 90),
    longitude     DOUBLE PRECISION NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    altitude_m    INTEGER          NOT NULL DEFAULT 0,
    speed_kmh     REAL             NOT NULL DEFAULT 0,
    direction_deg SMALLINT         NOT NULL DEFAULT 0,
    alarm_flags   BIGINT           NOT NULL DEFAULT 0,
    status_flags  BIGINT           NOT NULL DEFAULT 0,
    is_backfill   BOOLEAN          NOT NULL DEFAULT false,
    PRIMARY KEY (device_id, device_time)
);

-- +goose Down
DROP TABLE locations;