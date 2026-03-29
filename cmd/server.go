package cmd

import (
	"log"
	"net/http"

	"github.com/agentflow/agentflow-core/internal/bus"
	"github.com/agentflow/agentflow-core/internal/storage"
)

// RunServer starts the HTTP server with all routes registered
func RunServer(store *storage.SQLiteStore) (http.Handler, *bus.Bus) {
	mux := http.NewServeMux()

	// Initialize event bus
	eventBus := bus.New()

	// Register agent handlers
	agentHandler := NewAgentHandler(store, eventBus)
	agentHandler.RegisterRoutes(mux)

	log.Println("All routes registered")
	return mux, eventBus
}
