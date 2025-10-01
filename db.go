package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Task struct {
	ID           int64
	ChatID       int64
	CreatorID    int64
	CreatorName  string
	AssigneeID   int64
	AssigneeName string
	Text         string
	Done         bool
	CreatedAt    time.Time
}

func initDB(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec("PRAGMA busy_timeout = 5000;"); err != nil {
		db.Close()
		return nil, err
	}

	createTasks := `
	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER NOT NULL,
		creator_id INTEGER NOT NULL,
		creator_name TEXT,
		assignee_id INTEGER NOT NULL,
		assignee_name TEXT,
		text TEXT NOT NULL,
		done INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL
	);`
	if _, err := db.Exec(createTasks); err != nil {
		db.Close()
		return nil, err
	}

	createMembers := `
	CREATE TABLE IF NOT EXISTS members (
		chat_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		user_name TEXT,
		PRIMARY KEY(chat_id, user_id)
	);`
	if _, err := db.Exec(createMembers); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func SaveTask(db *sql.DB, task Task) (int64, error) {
	res, err := db.Exec(`
		INSERT INTO tasks (chat_id, creator_id, creator_name, assignee_id, assignee_name, text, done, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ChatID,
		task.CreatorID,
		task.CreatorName,
		task.AssigneeID,
		task.AssigneeName,
		task.Text,
		0, // новая задача всегда "не выполнена"
		task.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func GetTasksByUser(db *sql.DB, userID int64) ([]Task, error) {
	rows, err := db.Query(`
		SELECT id, chat_id, creator_id, creator_name, assignee_id, assignee_name, text, done, created_at
		FROM tasks
		WHERE assignee_id = ? AND done = 0
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var createdAt string
		var doneInt int
		if err := rows.Scan(
			&t.ID, &t.ChatID, &t.CreatorID, &t.CreatorName,
			&t.AssigneeID, &t.AssigneeName, &t.Text, &doneInt, &createdAt,
		); err != nil {
			return nil, err
		}
		t.Done = doneInt == 1
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		tasks = append(tasks, t)
	}
	return tasks, nil
}
