package api

import (
	"encoding/json"
	"fmt"
	"os"

	"everest.local/data-agent-policy-resolver/internal/execution"
)

func loadDefaultExecutionRequest() (*execution.ExecutionStartRequest, error) {
	path := "demo/execution/execution-standard.json"

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read default execution request: %w", err)
	}

	var req execution.ExecutionStartRequest
	if err := json.Unmarshal(b, &req); err != nil {
		return nil, fmt.Errorf("parse default execution request: %w", err)
	}

	return &req, nil
}
