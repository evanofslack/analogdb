CREATE TABLE IF NOT EXISTS ui_events (
    event_id       UUID,
    event_name     LowCardinality(String),
    schema_version UInt8,
    visitor_id     String,
    client_ts      DateTime64(3, 'UTC'),
    received_ts    DateTime64(3, 'UTC'),
    path           String,
    route          LowCardinality(String),
    referrer_host  LowCardinality(String),
    utm_source     LowCardinality(String),
    utm_medium     LowCardinality(String),
    utm_campaign   LowCardinality(String),
    viewport_width UInt16,
    device_type    LowCardinality(String),
    browser        LowCardinality(String),
    os             LowCardinality(String),
    country        LowCardinality(String),
    is_bot         Bool,
    search_id      String,
    post_id        UInt32,
    props          String
) ENGINE = ReplacingMergeTree
PARTITION BY toYYYYMM(received_ts)
ORDER BY (event_name, toDate(client_ts), event_id)
TTL toDateTime(received_ts) + INTERVAL 13 MONTH;
