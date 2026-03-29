package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/agentflow/agentflow-core/internal/bus"
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
	mux, eventBus := runServer(store)
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

// runServer starts the HTTP server with all routes registered
func runServer(store *storage.SQLiteStore) (http.Handler, *bus.Bus) {
	mux := http.NewServeMux()

	// Initialize event bus
	eventBus := bus.New()

	// Register agent handlers
	agentHandler := NewAgentHandler(store, eventBus)
	agentHandler.RegisterRoutes(mux)

	// Register task handlers
	taskHandler := NewTaskHandler(store, eventBus)
	taskHandler.RegisterRoutes(mux)

	// Register event handlers
	eventHandler := NewEventHandler(store, eventBus)
	eventHandler.RegisterRoutes(mux)

	log.Println("All routes registered")
	return mux, eventBus
}
