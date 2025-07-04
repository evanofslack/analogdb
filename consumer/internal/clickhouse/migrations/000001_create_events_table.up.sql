CREATE TABLE IF NOT EXISTS events (
    request_id String,
    remote_ip String,
    url String,
    path String,
    protocol String,
    scheme String,
    method String,
    user_agent String,
    response_code Int32,
    hostname String,
    authorized Bool,
    start_time Int64,
    end_time Int64,
    request_time_ms Int64,
    bytes_in Int32,
    bytes_out Int32,
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY (start_time, request_id)
PARTITION BY toYYYYMM(toDateTime(start_time / 1000));
