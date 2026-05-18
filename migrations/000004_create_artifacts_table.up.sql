CREATE TABLE artifacts (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id),
    task_id TEXT NOT NULL REFERENCES tasks(id),
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    data JSONB,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);
