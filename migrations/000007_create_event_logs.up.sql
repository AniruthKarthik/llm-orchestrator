CREATE TABLE IF NOT EXISTS event_logs (
    id VARCHAR(255) PRIMARY KEY,
    type TEXT NOT NULL,
    workflow_id VARCHAR(255) NOT NULL,
    task_id VARCHAR(255),
    severity TEXT NOT NULL,
    message TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX idx_event_logs_workflow_id ON event_logs(workflow_id);
CREATE INDEX idx_event_logs_timestamp ON event_logs(timestamp);
