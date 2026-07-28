package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-vanilla-crud/db"
	"go-vanilla-crud/middleware"
)

type Task struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`   // 'todo', 'in_progress', 'completed'
	Priority    string     `json:"priority"` // 'low', 'medium', 'high', 'urgent'
	Category    string     `json:"category"`
	DueDate     *time.Time `json:"due_date"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CreateTaskRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	Category    string  `json:"category"`
	DueDate     *string `json:"due_date"`
}

type UpdateTaskRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	Category    string  `json:"category"`
	DueDate     *string `json:"due_date"`
}

type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// GetTasksHandler retrieves all tasks for the logged in user with search & filtering
func GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	user, _ := middleware.GetUserFromContext(r.Context())

	statusFilter := r.URL.Query().Get("status")
	priorityFilter := r.URL.Query().Get("priority")
	categoryFilter := r.URL.Query().Get("category")
	searchQuery := r.URL.Query().Get("q")

	query := "SELECT id, user_id, title, COALESCE(description, ''), status, priority, category, due_date, created_at, updated_at FROM tasks WHERE user_id = $1"
	args := []interface{}{user.UserID}
	argID := 2

	if statusFilter != "" && statusFilter != "all" {
		query += " AND status = $" + strconv.Itoa(argID)
		args = append(args, statusFilter)
		argID++
	}

	if priorityFilter != "" && priorityFilter != "all" {
		query += " AND priority = $" + strconv.Itoa(argID)
		args = append(args, priorityFilter)
		argID++
	}

	if categoryFilter != "" && categoryFilter != "all" {
		query += " AND category = $" + strconv.Itoa(argID)
		args = append(args, categoryFilter)
		argID++
	}

	if searchQuery != "" {
		query += " AND (title ILIKE $" + strconv.Itoa(argID) + " OR description ILIKE $" + strconv.Itoa(argID) + ")"
		args = append(args, "%"+searchQuery+"%")
		argID++
	}

	query += " ORDER BY CASE status WHEN 'todo' THEN 1 WHEN 'in_progress' THEN 2 WHEN 'completed' THEN 3 END, due_date ASC NULLS LAST, created_at DESC"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch tasks"})
		return
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		var t Task
		var dueDate sql.NullTime
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.Category, &dueDate, &t.CreatedAt, &t.UpdatedAt); err != nil {
			continue
		}
		if dueDate.Valid {
			t.DueDate = &dueDate.Time
		}
		tasks = append(tasks, t)
	}

	json.NewEncoder(w).Encode(tasks)
}

// GetTaskByIDHandler fetches a single task by ID
func GetTaskByIDHandler(w http.ResponseWriter, r *http.Request, taskID int) {
	w.Header().Set("Content-Type", "application/json")
	user, _ := middleware.GetUserFromContext(r.Context())

	var t Task
	var dueDate sql.NullTime
	err := db.DB.QueryRow(
		"SELECT id, user_id, title, COALESCE(description, ''), status, priority, category, due_date, created_at, updated_at FROM tasks WHERE id = $1 AND user_id = $2",
		taskID, user.UserID,
	).Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.Category, &dueDate, &t.CreatedAt, &t.UpdatedAt)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch task"})
		return
	}

	if dueDate.Valid {
		t.DueDate = &dueDate.Time
	}

	json.NewEncoder(w).Encode(t)
}

// CreateTaskHandler creates a new task
func CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user, _ := middleware.GetUserFromContext(r.Context())

	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Title is required"})
		return
	}

	if req.Status == "" {
		req.Status = "todo"
	}
	if req.Priority == "" {
		req.Priority = "medium"
	}
	if req.Category == "" {
		req.Category = "General"
	}

	var parsedDueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		t, err := time.Parse(time.RFC3339, *req.DueDate)
		if err == nil {
			parsedDueDate = &t
		} else {
			// Try YYYY-MM-DD
			t, err = time.Parse("2006-01-02", *req.DueDate)
			if err == nil {
				parsedDueDate = &t
			}
		}
	}

	var task Task
	var dueDate sql.NullTime
	err := db.DB.QueryRow(
		`INSERT INTO tasks (user_id, title, description, status, priority, category, due_date)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, user_id, title, COALESCE(description, ''), status, priority, category, due_date, created_at, updated_at`,
		user.UserID, req.Title, req.Description, req.Status, req.Priority, req.Category, parsedDueDate,
	).Scan(&task.ID, &task.UserID, &task.Title, &task.Description, &task.Status, &task.Priority, &task.Category, &dueDate, &task.CreatedAt, &task.UpdatedAt)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create task"})
		return
	}

	if dueDate.Valid {
		task.DueDate = &dueDate.Time
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

// UpdateTaskHandler updates an existing task
func UpdateTaskHandler(w http.ResponseWriter, r *http.Request, taskID int) {
	w.Header().Set("Content-Type", "application/json")
	user, _ := middleware.GetUserFromContext(r.Context())

	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Title is required"})
		return
	}

	var parsedDueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		t, err := time.Parse(time.RFC3339, *req.DueDate)
		if err == nil {
			parsedDueDate = &t
		} else {
			t, err = time.Parse("2006-01-02", *req.DueDate)
			if err == nil {
				parsedDueDate = &t
			}
		}
	}

	var task Task
	var dueDate sql.NullTime
	err := db.DB.QueryRow(
		`UPDATE tasks
		 SET title = $1, description = $2, status = $3, priority = $4, category = $5, due_date = $6, updated_at = NOW()
		 WHERE id = $7 AND user_id = $8
		 RETURNING id, user_id, title, COALESCE(description, ''), status, priority, category, due_date, created_at, updated_at`,
		req.Title, req.Description, req.Status, req.Priority, req.Category, parsedDueDate, taskID, user.UserID,
	).Scan(&task.ID, &task.UserID, &task.Title, &task.Description, &task.Status, &task.Priority, &task.Category, &dueDate, &task.CreatedAt, &task.UpdatedAt)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update task"})
		return
	}

	if dueDate.Valid {
		task.DueDate = &dueDate.Time
	}

	json.NewEncoder(w).Encode(task)
}

// UpdateTaskStatusHandler quickly updates task status
func UpdateTaskStatusHandler(w http.ResponseWriter, r *http.Request, taskID int) {
	w.Header().Set("Content-Type", "application/json")
	user, _ := middleware.GetUserFromContext(r.Context())

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Status == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Valid status is required"})
		return
	}

	var task Task
	var dueDate sql.NullTime
	err := db.DB.QueryRow(
		`UPDATE tasks
		 SET status = $1, updated_at = NOW()
		 WHERE id = $2 AND user_id = $3
		 RETURNING id, user_id, title, COALESCE(description, ''), status, priority, category, due_date, created_at, updated_at`,
		req.Status, taskID, user.UserID,
	).Scan(&task.ID, &task.UserID, &task.Title, &task.Description, &task.Status, &task.Priority, &task.Category, &dueDate, &task.CreatedAt, &task.UpdatedAt)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update status"})
		return
	}

	if dueDate.Valid {
		task.DueDate = &dueDate.Time
	}

	json.NewEncoder(w).Encode(task)
}

// DeleteTaskHandler deletes a task
func DeleteTaskHandler(w http.ResponseWriter, r *http.Request, taskID int) {
	w.Header().Set("Content-Type", "application/json")
	user, _ := middleware.GetUserFromContext(r.Context())

	res, err := db.DB.Exec("DELETE FROM tasks WHERE id = $1 AND user_id = $2", taskID, user.UserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete task"})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Task deleted successfully"})
}

// TaskRouter handles /api/tasks and /api/tasks/{id}
func TaskRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		switch r.Method {
		case http.MethodGet:
			GetTasksHandler(w, r)
		case http.MethodPost:
			CreateTaskHandler(w, r)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	parts := strings.Split(path, "/")
	taskID, err := strconv.Atoi(parts[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid task ID"})
		return
	}

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			GetTaskByIDHandler(w, r, taskID)
		case http.MethodPut:
			UpdateTaskHandler(w, r, taskID)
		case http.MethodDelete:
			DeleteTaskHandler(w, r, taskID)
		default:
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	if len(parts) == 2 && parts[1] == "status" && r.Method == http.MethodPatch {
		UpdateTaskStatusHandler(w, r, taskID)
		return
	}

	http.Error(w, `{"error":"Not found"}`, http.StatusNotFound)
}
