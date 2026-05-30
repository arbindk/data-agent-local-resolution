package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"everest.local/data-agent-policy-resolver/internal/api"
	"everest.local/data-agent-policy-resolver/internal/config"
	"everest.local/data-agent-policy-resolver/internal/policyresolver"
)

func main() {
	addr := ":8080"
	if v := os.Getenv("POLICY_AGENT_ADDR"); v != "" {
		addr = v
	}

	cfg := config.Load()
	resolver := policyresolver.NewResolver(cfg)
	server := api.NewServer(resolver)

	fmt.Println("Data Agent Local Resolution service starting on", addr)
	log.Fatal(http.ListenAndServe(addr, server.Routes()))
}
