package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"everest.local/data-agent-policy-resolver/internal/agentruntime"
	"everest.local/data-agent-policy-resolver/internal/api"
	"everest.local/data-agent-policy-resolver/internal/config"
	"everest.local/data-agent-policy-resolver/internal/execution"
	"everest.local/data-agent-policy-resolver/internal/localstore/duckdbstore"
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
	workflowResolver := execution.NewWorkflowResolver(cfg.WorkflowRepoPath)

	duckStore, err := duckdbstore.Open(cfg.DuckDBPath)
	if err != nil {
		log.Fatal("failed to open DuckDB store: ", err)
	}
	defer duckStore.Close()

	orchestrator := execution.NewOrchestrator(executionManager, policyResolver, workflowResolver, duckStore)

	server := api.NewServer(cfg, policyResolver, orchestrator, duckStore)

	log.Printf("Everest Data Agent service starting on %s", addr)

	pollEnabled := strings.EqualFold(os.Getenv("EVEREST_AGENT_POLL_ENABLED"), "true")

	pollIntervalSeconds := 15
	if raw := os.Getenv("EVEREST_AGENT_POLL_INTERVAL_SECONDS"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			pollIntervalSeconds = parsed
		}
	}

	pollTargetURL := agentBaseURL(addr) + "/v1/controlplane/run-next-lease"

	agentruntime.ConfigurePolling(
		pollEnabled,
		pollIntervalSeconds,
		pollTargetURL,
	)

	if pollEnabled {
		go startControlPlaneLeasePoller(
			context.Background(),
			addr,
			time.Duration(pollIntervalSeconds)*time.Second,
		)

		log.Printf("Control Plane lease polling enabled; interval=%s", time.Duration(pollIntervalSeconds)*time.Second)
	} else {
		log.Printf("Control Plane lease polling disabled; set EVEREST_AGENT_POLL_ENABLED=true to enable")
	}

	if err := http.ListenAndServe(addr, server.Routes()); err != nil {
		log.Fatal(err)
	}
}

func startControlPlaneLeasePoller(ctx context.Context, addr string, interval time.Duration) {
	if interval <= 0 {
		interval = 15 * time.Second
	}

	baseURL := agentBaseURL(addr)

	client := &http.Client{
		Timeout: 10 * time.Minute,
	}

	log.Printf("Control Plane lease poller targeting %s/v1/controlplane/run-next-lease", baseURL)

	// Give the local HTTP server a few seconds to start before the first poll.
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		log.Printf("Control Plane lease polling stopped before first poll")
		return
	case <-timer.C:
		pollOnceForLease(ctx, client, baseURL)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Control Plane lease polling stopped")
			return

		case <-ticker.C:
			pollOnceForLease(ctx, client, baseURL)
		}
	}
}

func pollOnceForLease(ctx context.Context, client *http.Client, baseURL string) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		baseURL+"/v1/controlplane/run-next-lease",
		bytes.NewReader([]byte("{}")),
	)
	if err != nil {
		log.Printf("Control Plane lease polling request build failed: %v", err)
		agentruntime.RecordPollResult(0, "failed", err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		agentruntime.RecordPollResult(0, "failed", err.Error())
		log.Printf("Control Plane lease polling failed: %v", err)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	message := string(bodyBytes)
	if len(message) > 1000 {
		message = message[:1000]
	}

	switch {
	case resp.StatusCode >= 500:
		agentruntime.RecordPollResult(resp.StatusCode, "server_error", message)
		log.Printf("Control Plane lease polling returned HTTP %d", resp.StatusCode)

	case resp.StatusCode >= 400:
		agentruntime.RecordPollResult(resp.StatusCode, "client_error", message)
		log.Printf("Control Plane lease polling returned HTTP %d", resp.StatusCode)

	default:
		agentruntime.RecordPollResult(resp.StatusCode, "ok", message)
		log.Printf("Control Plane lease polling completed with HTTP %d", resp.StatusCode)
	}
}

func agentBaseURL(addr string) string {
	addr = strings.TrimSpace(addr)

	if addr == "" || addr == ":8080" {
		return "http://localhost:8080"
	}

	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr
	}

	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}

	return "http://" + strings.TrimRight(addr, "/")
}
