CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE tasks(
    id SERIAL PRIMARY KEY,
    task_id TEXT NOT NULL DEFAULT uuid_generate_v1()::text,
    category TEXT NOT NULL,
    err TEXT,
    result TEXT,
    payload TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP
);
