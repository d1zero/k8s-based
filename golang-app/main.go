package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Todo represents a single todo item
type Todo struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// JSONResponse is a generic structure for API responses
type JSONResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// App holds the application state and dependencies
type App struct {
	db *sql.DB
	mu sync.Mutex
}

// writeJSONResponse is a helper function to write JSON responses
func writeJSONResponse(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp := JSONResponse{Message: message, Data: data}
	json.NewEncoder(w).Encode(resp)
}

// handleError is a helper function to handle errors
func handleError(w http.ResponseWriter, err error, message string, statusCode int) {
	log.Printf("Error: %s, Details: %v", message, err)
	writeJSONResponse(w, statusCode, message, nil)
}

// initDB initializes the PostgreSQL database connection
func initDB() (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"))

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create todos table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS todos (
			id VARCHAR(36) PRIMARY KEY,
			title TEXT NOT NULL,
			completed BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create todos table: %w", err)
	}

	return db, nil
}

// getTodosHandler handles GET requests to /todos
func (app *App) getTodosHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := app.db.Query("SELECT id, title, completed, created_at, updated_at FROM todos")
	if err != nil {
		handleError(w, err, "Failed to retrieve todos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var todo Todo
		if err := rows.Scan(&todo.ID, &todo.Title, &todo.Completed, &todo.CreatedAt, &todo.UpdatedAt); err != nil {
			handleError(w, err, "Failed to scan todo", http.StatusInternalServerError)
			return
		}
		todos = append(todos, todo)
	}

	writeJSONResponse(w, http.StatusOK, "Todos retrieved successfully", todos)
}

// createTodoHandler handles POST requests to /todos
func (app *App) createTodoHandler(w http.ResponseWriter, r *http.Request) {
	var newTodo Todo
	if err := json.NewDecoder(r.Body).Decode(&newTodo); err != nil {
		handleError(w, err, "Invalid request payload", http.StatusBadRequest)
		return
	}

	newTodo.ID = uuid.New().String()
	newTodo.Completed = false

	_, err := app.db.Exec(
		"INSERT INTO todos (id, title, completed) VALUES ($1, $2, $3)",
		newTodo.ID, newTodo.Title, newTodo.Completed)
	if err != nil {
		handleError(w, err, "Failed to create todo", http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, http.StatusCreated, "Todo created successfully", newTodo)
}

// getTodoByIDHandler handles GET requests to /todos/{id}
func (app *App) getTodoByIDHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/todos/"):] // Extract ID from URL path

	var todo Todo
	err := app.db.QueryRow(
		"SELECT id, title, completed, created_at, updated_at FROM todos WHERE id = $1", id).
		Scan(&todo.ID, &todo.Title, &todo.Completed, &todo.CreatedAt, &todo.UpdatedAt)
	if err == sql.ErrNoRows {
		writeJSONResponse(w, http.StatusNotFound, "Todo not found", nil)
		return
	} else if err != nil {
		handleError(w, err, "Failed to retrieve todo", http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, http.StatusOK, "Todo retrieved successfully", todo)
}

// updateTodoHandler handles PUT requests to /todos/{id}
func (app *App) updateTodoHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/todos/"):] // Extract ID from URL path

	var updatedTodo Todo
	if err := json.NewDecoder(r.Body).Decode(&updatedTodo); err != nil {
		handleError(w, err, "Invalid request payload", http.StatusBadRequest)
		return
	}

	result, err := app.db.Exec(
		"UPDATE todos SET title = $1, completed = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3",
		updatedTodo.Title, updatedTodo.Completed, id)
	if err != nil {
		handleError(w, err, "Failed to update todo", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		handleError(w, err, "Failed to check update result", http.StatusInternalServerError)
		return
	}
	if rowsAffected == 0 {
		writeJSONResponse(w, http.StatusNotFound, "Todo not found", nil)
		return
	}

	// Fetch the updated todo
	var todo Todo
	err = app.db.QueryRow(
		"SELECT id, title, completed, created_at, updated_at FROM todos WHERE id = $1", id).
		Scan(&todo.ID, &todo.Title, &todo.Completed, &todo.CreatedAt, &todo.UpdatedAt)
	if err != nil {
		handleError(w, err, "Failed to retrieve updated todo", http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, http.StatusOK, "Todo updated successfully", todo)
}

// deleteTodoHandler handles DELETE requests to /todos/{id}
func (app *App) deleteTodoHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/todos/"):] // Extract ID from URL path

	result, err := app.db.Exec("DELETE FROM todos WHERE id = $1", id)
	if err != nil {
		handleError(w, err, "Failed to delete todo", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		handleError(w, err, "Failed to check delete result", http.StatusInternalServerError)
		return
	}
	if rowsAffected == 0 {
		writeJSONResponse(w, http.StatusNotFound, "Todo not found", nil)
		return
	}

	writeJSONResponse(w, http.StatusNoContent, "Todo deleted successfully", nil)
}

// healthCheckHandler handles GET requests to /health
func (app *App) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	if err := app.db.Ping(); err != nil {
		handleError(w, err, "Database connection failed", http.StatusInternalServerError)
		return
	}

	status := struct {
		Status    string `json:"status"`
		Timestamp string `json:"timestamp"`
		Message   string `json:"message"`
	}{
		Status:    "UP",
		Timestamp: time.Now().Format(time.RFC3339),
		Message:   "API and database are healthy and running",
	}
	writeJSONResponse(w, http.StatusOK, "Health Check", status)
}

func main() {
	// Initialize database
	db, err := initDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create app instance
	app := &App{db: db}

	// Define API routes
	http.HandleFunc("/todos", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			app.getTodosHandler(w, r)
		case http.MethodPost:
			app.createTodoHandler(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/todos/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			app.getTodoByIDHandler(w, r)
		case http.MethodPut:
			app.updateTodoHandler(w, r)
		case http.MethodDelete:
			app.deleteTodoHandler(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/health", app.healthCheckHandler)

	fmt.Println("Server listening on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
