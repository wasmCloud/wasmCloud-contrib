CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE tasks(
    id SERIAL PRIMARY KEY,
    task_id TEXT NOT NULL DEFAULT uuid_generate_v1()::text,
    original_asset TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),

    resize_error TEXT,
    resized_asset TEXT,
    resized_at TIMESTAMP,

    analyze_error TEXT,
    analyze_result BOOL,
    analyzed_at TIMESTAMP
);
