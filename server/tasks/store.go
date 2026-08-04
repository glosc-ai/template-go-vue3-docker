package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("task not found")

type SQLStore struct {
	db     *sql.DB
	driver string
}

func NewSQLStore(db *sql.DB, driver string) *SQLStore {
	return &SQLStore{db: db, driver: driver}
}

func (s *SQLStore) List(ctx context.Context) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, completed, created_at
		FROM tasks
		ORDER BY created_at DESC, id DESC
		LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("listing tasks: %w", err)
	}
	defer rows.Close()

	items := make([]Task, 0)
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.Title, &task.Completed, &task.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning task: %w", err)
		}
		items = append(items, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading tasks: %w", err)
	}
	return items, nil
}

func (s *SQLStore) Create(ctx context.Context, title string) (Task, error) {
	if s.driver == "postgres" {
		var task Task
		err := s.db.QueryRowContext(ctx, `
			INSERT INTO tasks (title)
			VALUES ($1)
			RETURNING id, title, completed, created_at`, title).
			Scan(&task.ID, &task.Title, &task.Completed, &task.CreatedAt)
		if err != nil {
			return Task{}, fmt.Errorf("creating task: %w", err)
		}
		return task, nil
	}

	result, err := s.db.ExecContext(ctx, "INSERT INTO tasks (title) VALUES (?)", title)
	if err != nil {
		return Task{}, fmt.Errorf("creating task: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Task{}, fmt.Errorf("reading created task ID: %w", err)
	}
	return s.Lookup(ctx, id)
}

func (s *SQLStore) Lookup(ctx context.Context, id int64) (Task, error) {
	query := "SELECT id, title, completed, created_at FROM tasks WHERE id = ?"
	if s.driver == "postgres" {
		query = "SELECT id, title, completed, created_at FROM tasks WHERE id = $1"
	}
	var task Task
	if err := s.db.QueryRowContext(ctx, query, id).Scan(&task.ID, &task.Title, &task.Completed, &task.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, ErrNotFound
		}
		return Task{}, fmt.Errorf("looking up task %d: %w", id, err)
	}
	return task, nil
}

func (s *SQLStore) SetCompleted(ctx context.Context, id int64, completed bool) (Task, error) {
	query := "UPDATE tasks SET completed = ? WHERE id = ?"
	args := []any{completed, id}
	if s.driver == "postgres" {
		query = "UPDATE tasks SET completed = $1 WHERE id = $2"
	}
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return Task{}, fmt.Errorf("updating task %d: %w", id, err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return Task{}, fmt.Errorf("reading updated row count: %w", err)
	}
	if updated == 0 {
		return Task{}, ErrNotFound
	}
	return s.Lookup(ctx, id)
}

func (s *SQLStore) Delete(ctx context.Context, id int64) error {
	query := "DELETE FROM tasks WHERE id = ?"
	if s.driver == "postgres" {
		query = "DELETE FROM tasks WHERE id = $1"
	}
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting task %d: %w", id, err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("reading deleted row count: %w", err)
	}
	if deleted == 0 {
		return ErrNotFound
	}
	return nil
}
