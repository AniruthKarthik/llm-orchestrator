#!/bin/bash
expect <<EOD
set timeout 60
spawn go run ./cmd/orch workflow.yaml.example
expect "[PROMPT] Task 'summary-task' is waiting for approval. Approve? (y/n):"
send "y\r"
expect "Execution completed successfully!"
expect eof
EOD
