CREATE TABLE agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    role TEXT NOT NULL,
    system_prompt TEXT,
    model TEXT NOT NULL,
    provider TEXT NOT NULL,
    tools JSONB,
    config JSONB
);

ALTER TABLE tasks ADD COLUMN agent_id TEXT;
