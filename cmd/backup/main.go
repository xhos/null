package main

import (
	"context"
	"encoding/json"
	"fmt"
	"null/internal/backup"
	"null/internal/config"
	"null/internal/db"
	"os"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "backup":
		runBackup()
	case "restore":
		runRestore()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage:
  null backup <file>
  null restore <file>

Environment:
  DATABASE_URL   PostgreSQL connection string
  USER_ID        User UUID`)
}

func runBackup() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: null backup <file>")
		os.Exit(1)
	}

	filename := os.Args[2]
	cfg := config.Load()
	userID := getUserID()

	store, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("database connection failed", "err", err)
	}
	defer store.Close()

	ctx := context.Background()
	data, err := backup.ExportAll(ctx, store.Queries, userID)
	if err != nil {
		log.Fatal("backup failed", "err", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		log.Fatal("failed to create file", "file", filename, "err", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		log.Fatal("failed to write backup", "err", err)
	}

	fmt.Printf("Backup created successfully: %s\n", filename)
	fmt.Printf("  Categories:   %d\n", len(data.Categories))
	fmt.Printf("  Accounts:     %d\n", len(data.Accounts))
	fmt.Printf("  Transactions: %d\n", len(data.Transactions))
	fmt.Printf("  Rules:        %d\n", len(data.Rules))
}

func runRestore() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: null restore <file>")
		os.Exit(1)
	}

	filename := os.Args[2]
	cfg := config.Load()
	userID := getUserID()

	store, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("database connection failed", "err", err)
	}
	defer store.Close()

	file, err := os.Open(filename)
	if err != nil {
		log.Fatal("failed to open file", "file", filename, "err", err)
	}
	defer file.Close()

	var data backup.Backup
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		log.Fatal("failed to parse backup file", "err", err)
	}

	ctx := context.Background()
	if err := backup.ImportAll(ctx, store.Queries, userID, &data); err != nil {
		log.Fatal("restore failed", "err", err)
	}

	fmt.Printf("Restore completed successfully\n")
	fmt.Printf("  Categories:   %d\n", len(data.Categories))
	fmt.Printf("  Accounts:     %d\n", len(data.Accounts))
	fmt.Printf("  Transactions: %d\n", len(data.Transactions))
	fmt.Printf("  Rules:        %d\n", len(data.Rules))
}

func getUserID() uuid.UUID {
	userIDStr := os.Getenv("USER_ID")
	if userIDStr == "" {
		log.Fatal("USER_ID environment variable is required")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Fatal("invalid USER_ID", "err", err)
	}

	return userID
}
