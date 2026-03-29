package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/agentflow/agentflow-core/cmd"
	"github.com/agentflow/agentflow-core/internal/storage"
)

func main() {
	fmt.Println("agentflow-core starting...")

	// Initialize SQLite store
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/agentflow.db"
	}

	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer store.Close()

	// Register routes and get event bus
	mux, eventBus := cmd.RunServer(store)
	defer eventBus.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s\n", port)
	log.Printf("agentflow-core is running on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
