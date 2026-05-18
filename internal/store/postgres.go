package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, connString string, migrationsPath string) (*PostgresStore, error) {
	// Run migrations
	if migrationsPath != "" {
		m, err := migrate.New(
			fmt.Sprintf("file://%s", migrationsPath),
			connString,
		)
		if err != nil {
			return nil, fmt.Errorf("could not create migrate instance: %w", err)
		}
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return nil, fmt.Errorf("could not run migrations: %w", err)
		}
	}

	// Create connection pool
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("could not parse connection string: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("could not create connection pool: %w", err)
	}

	// Ping to verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("could not ping database: %w", err)
	}

	return &PostgresStore{
		pool: pool,
	}, nil
}

func (s *PostgresStore) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *PostgresStore) SaveWorkflow(workflow WorkflowRecord) error {
	ctx := context.Background()
	query := `
		INSERT INTO workflows (id, name, description, status, created_at, started_at, finished_at, timeout)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := s.pool.Exec(ctx, query,
		workflow.ID,
		workflow.Name,
		workflow.Description,
		workflow.Status,
		workflow.CreatedAt,
		workflow.StartedAt,
		workflow.FinishedAt,
		int64(workflow.Timeout),
	)
	if err != nil {
		return fmt.Errorf("failed to save workflow: %w", err)
	}
	return nil
}

func (s *PostgresStore) UpdateWorkflow(workflow WorkflowRecord) error {
	ctx := context.Background()
	query := `
		UPDATE workflows
		SET name = $2, description = $3, status = $4, started_at = $5, finished_at = $6, timeout = $7
		WHERE id = $1
	`
	tag, err := s.pool.Exec(ctx, query,
		workflow.ID,
		workflow.Name,
		workflow.Description,
		workflow.Status,
		workflow.StartedAt,
		workflow.FinishedAt,
		int64(workflow.Timeout),
	)
	if err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("workflow not found: %s", workflow.ID)
	}
	return nil
}

func (s *PostgresStore) GetWorkflow(workflowID string) (WorkflowRecord, error) {
	ctx := context.Background()
	query := `
		SELECT id, name, description, status, created_at, started_at, finished_at, timeout
		FROM workflows
		WHERE id = $1
	`
	var w WorkflowRecord
	var timeout int64
	err := s.pool.QueryRow(ctx, query, workflowID).Scan(
		&w.ID,
		&w.Name,
		&w.Description,
		&w.Status,
		&w.CreatedAt,
		&w.StartedAt,
		&w.FinishedAt,
		&timeout,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WorkflowRecord{}, fmt.Errorf("workflow not found: %s", workflowID)
		}
		return WorkflowRecord{}, fmt.Errorf("failed to get workflow: %w", err)
	}
	w.Timeout = time.Duration(timeout)
	return w, nil
}

func (s *PostgresStore) SaveTask(task TaskRecord) error {
	ctx := context.Background()
	inputJSON, _ := json.Marshal(task.Input)
	outputJSON, _ := json.Marshal(task.Output)

	query := `
		INSERT INTO tasks (id, workflow_id, name, description, status, error, input, output, dependencies, created_at, started_at, finished_at, timeout)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := s.pool.Exec(ctx, query,
		task.ID,
		task.WorkflowID,
		task.Name,
		task.Description,
		task.Status,
		task.Error,
		inputJSON,
		outputJSON,
		task.Dependencies,
		task.CreatedAt,
		task.StartedAt,
		task.FinishedAt,
		int64(task.Timeout),
	)
	if err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}
	return nil
}

func (s *PostgresStore) UpdateTask(task TaskRecord) error {
	ctx := context.Background()
	inputJSON, _ := json.Marshal(task.Input)
	outputJSON, _ := json.Marshal(task.Output)

	query := `
		UPDATE tasks
		SET name = $3, description = $4, status = $5, error = $6, input = $7, output = $8, dependencies = $9, started_at = $10, finished_at = $11, timeout = $12
		WHERE id = $1 AND workflow_id = $2
	`
	tag, err := s.pool.Exec(ctx, query,
		task.ID,
		task.WorkflowID,
		task.Name,
		task.Description,
		task.Status,
		task.Error,
		inputJSON,
		outputJSON,
		task.Dependencies,
		task.StartedAt,
		task.FinishedAt,
		int64(task.Timeout),
	)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("task not found: %s in workflow %s", task.ID, task.WorkflowID)
	}
	return nil
}

func (s *PostgresStore) GetTask(workflowID string, taskID string) (TaskRecord, error) {
	ctx := context.Background()
	query := `
		SELECT id, workflow_id, name, description, status, error, input, output, dependencies, created_at, started_at, finished_at, timeout
		FROM tasks
		WHERE id = $1 AND workflow_id = $2
	`
	var t TaskRecord
	var inputJSON, outputJSON []byte
	var timeout int64
	err := s.pool.QueryRow(ctx, query, taskID, workflowID).Scan(
		&t.ID,
		&t.WorkflowID,
		&t.Name,
		&t.Description,
		&t.Status,
		&t.Error,
		&inputJSON,
		&outputJSON,
		&t.Dependencies,
		&t.CreatedAt,
		&t.StartedAt,
		&t.FinishedAt,
		&timeout,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TaskRecord{}, fmt.Errorf("task not found: %s", taskID)
		}
		return TaskRecord{}, fmt.Errorf("failed to get task: %w", err)
	}
	json.Unmarshal(inputJSON, &t.Input)
	json.Unmarshal(outputJSON, &t.Output)
	t.Timeout = time.Duration(timeout)
	return t, nil
}

func (s *PostgresStore) GetWorkflowTasks(workflowID string) ([]TaskRecord, error) {
	ctx := context.Background()
	query := `
		SELECT id, workflow_id, name, description, status, error, input, output, dependencies, created_at, started_at, finished_at, timeout
		FROM tasks
		WHERE workflow_id = $1
	`
	rows, err := s.pool.Query(ctx, query, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow tasks: %w", err)
	}
	defer rows.Close()

	var tasks []TaskRecord
	for rows.Next() {
		var t TaskRecord
		var inputJSON, outputJSON []byte
		var timeout int64
		err := rows.Scan(
			&t.ID,
			&t.WorkflowID,
			&t.Name,
			&t.Description,
			&t.Status,
			&t.Error,
			&inputJSON,
			&outputJSON,
			&t.Dependencies,
			&t.CreatedAt,
			&t.StartedAt,
			&t.FinishedAt,
			&timeout,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		json.Unmarshal(inputJSON, &t.Input)
		json.Unmarshal(outputJSON, &t.Output)
		t.Timeout = time.Duration(timeout)
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (s *PostgresStore) SaveCheckpoint(checkpoint CheckpointRecord) error {
	ctx := context.Background()
	query := `
		INSERT INTO checkpoints (workflow_id, state_data, timestamp)
		VALUES ($1, $2, $3)
	`
	_, err := s.pool.Exec(ctx, query,
		checkpoint.WorkflowID,
		checkpoint.StateData,
		checkpoint.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetLatestCheckpoint(workflowID string) (CheckpointRecord, error) {
	ctx := context.Background()
	query := `
		SELECT workflow_id, state_data, timestamp
		FROM checkpoints
		WHERE workflow_id = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`
	var cp CheckpointRecord
	err := s.pool.QueryRow(ctx, query, workflowID).Scan(
		&cp.WorkflowID,
		&cp.StateData,
		&cp.Timestamp,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CheckpointRecord{}, fmt.Errorf("no checkpoints found for workflow: %s", workflowID)
		}
		return CheckpointRecord{}, fmt.Errorf("failed to get latest checkpoint: %w", err)
	}
	return cp, nil
}

func (s *PostgresStore) ListWorkflows() ([]WorkflowRecord, error) {
	ctx := context.Background()
	query := `
		SELECT id, name, description, status, created_at, started_at, finished_at, timeout
		FROM workflows
	`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer rows.Close()

	var workflows []WorkflowRecord
	for rows.Next() {
		var w WorkflowRecord
		var timeout int64
		err := rows.Scan(
			&w.ID,
			&w.Name,
			&w.Description,
			&w.Status,
			&w.CreatedAt,
			&w.StartedAt,
			&w.FinishedAt,
			&timeout,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow: %w", err)
		}
		w.Timeout = time.Duration(timeout)
		workflows = append(workflows, w)
	}
	return workflows, nil
}
