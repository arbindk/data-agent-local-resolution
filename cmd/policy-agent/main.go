package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"everest.local/data-agent-policy-resolver/internal/api"
	"everest.local/data-agent-policy-resolver/internal/config"
	"everest.local/data-agent-policy-resolver/internal/execution"
	"everest.local/data-agent-policy-resolver/internal/policyresolver"
)

func main() {
	addr := ":8080"
	if v := os.Getenv("POLICY_AGENT_ADDR"); v != "" {
		addr = v
	}

	cfg := config.Load()

	policyResolver := policyresolver.NewResolver(cfg)
	executionManager := execution.NewManager()
	orchestrator := execution.NewOrchestrator(executionManager, policyResolver)

	server := api.NewServer(policyResolver, orchestrator)

	fmt.Println("Data Agent / Agent Core demo service starting on", addr)
	log.Fatal(http.ListenAndServe(addr, server.Routes()))
}
