package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	fmt.Println("agentflow-core starting...")

	// TODO: Initialize application
	// - Load configuration
	// - Initialize storage
	// - Start HTTP server

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server listening on port %s\n", port)
	log.Printf("agentflow-core is running on :%s", port)
}
