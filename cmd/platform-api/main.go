package main

import (
	"log"
	"net/http"
	"os"

	"everest.local/data-agent-policy-resolver/internal/platform"
)

func main() {
	addr := os.Getenv("EVEREST_PLATFORM_ADDR")
	if addr == "" {
		addr = ":9090"
	}

	dbPath := os.Getenv("EVEREST_PLATFORM_DB_PATH")
	if dbPath == "" {
		dbPath = "data/platform/everest-platform.sqlite"
	}

	store, err := platform.NewStore(dbPath)
	if err != nil {
		log.Fatal("failed to initialize Everest Platform store: ", err)
	}

	server := platform.NewServer(store)

	log.Printf("Everest Platform API starting on %s", addr)
	log.Printf("Everest Platform persistent store: %s", dbPath)

	if err := http.ListenAndServe(addr, server.Routes()); err != nil {
		log.Fatal(err)
	}
}
