package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() *sql.DB {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5432"
	}
	if user == "" {
		user = "postgres"
	}
	if password == "" {
		password = "postgres"
	}
	if dbname == "" {
		dbname = "taskdb"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var database *sql.DB
	var err error

	// Retry database connection for up to 30 seconds
	for i := 1; i <= 15; i++ {
		database, err = sql.Open("postgres", connStr)
		if err == nil {
			err = database.Ping()
			if err == nil {
				log.Println("Successfully connected to PostgreSQL database")
				break
			}
		}
		log.Printf("Waiting for database connection (attempt %d/15)...", i)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	database.SetMaxOpenConns(25)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(5 * time.Minute)

	DB = database
	runMigrations(database)

	return DB
}

func runMigrations(database *sql.DB) {
	initSQL, err := os.ReadFile("db/init.sql")
	if err != nil {
		// Fallback to embedded schema if file path differs
		log.Printf("Could not read db/init.sql, running inline schema migration: %v", err)
		initSQL = []byte(`
			CREATE TABLE IF NOT EXISTS users (
				id SERIAL PRIMARY KEY,
				username VARCHAR(50) NOT NULL UNIQUE,
				email VARCHAR(255) NOT NULL UNIQUE,
				password_hash VARCHAR(255) NOT NULL,
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);

			CREATE TABLE IF NOT EXISTS refresh_tokens (
				id SERIAL PRIMARY KEY,
				user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				token_hash VARCHAR(255) NOT NULL UNIQUE,
				expires_at TIMESTAMPTZ NOT NULL,
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);

			CREATE TABLE IF NOT EXISTS tasks (
				id SERIAL PRIMARY KEY,
				user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				title VARCHAR(255) NOT NULL,
				description TEXT,
				status VARCHAR(20) NOT NULL DEFAULT 'todo',
				priority VARCHAR(10) NOT NULL DEFAULT 'medium',
				category VARCHAR(50) NOT NULL DEFAULT 'General',
				due_date TIMESTAMPTZ,
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);
		`)
	}

	_, err = database.Exec(string(initSQL))
	if err != nil {
		log.Fatalf("Failed to run schema migrations: %v", err)
	}
	log.Println("Database schema initialized successfully")
}
