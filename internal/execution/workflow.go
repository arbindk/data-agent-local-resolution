package execution

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type WorkflowDefinition struct {
	WorkflowID      string         `json:"workflow_id"`
	WorkflowVersion string         `json:"workflow_version"`
	Description     string         `json:"description"`
	Steps           []WorkflowStep `json:"steps"`
}

type WorkflowStep struct {
	Name        string         `json:"name"`
	Owner       string         `json:"owner"`
	Required    bool           `json:"required"`
	Description string         `json:"description"`
	Config      map[string]any `json:"config,omitempty"`
}

type WorkflowResolver struct {
	WorkflowRepoPath string
}

func NewWorkflowResolver(workflowRepoPath string) *WorkflowResolver {
	return &WorkflowResolver{
		WorkflowRepoPath: workflowRepoPath,
	}
}

func (r *WorkflowResolver) Resolve(workflowVersion string) (*WorkflowDefinition, error) {
	if workflowVersion == "" {
		return nil, fmt.Errorf("workflow_version is required")
	}

	path := filepath.Join(r.WorkflowRepoPath, workflowVersion+".json")

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read workflow definition %s: %w", path, err)
	}

	var workflow WorkflowDefinition
	if err := json.Unmarshal(b, &workflow); err != nil {
		return nil, fmt.Errorf("parse workflow definition: %w", err)
	}

	if workflow.WorkflowVersion == "" {
		return nil, fmt.Errorf("workflow_version missing in workflow definition")
	}

	if workflow.WorkflowVersion != workflowVersion {
		return nil, fmt.Errorf("workflow version mismatch: requested=%s actual=%s", workflowVersion, workflow.WorkflowVersion)
	}

	if len(workflow.Steps) == 0 {
		return nil, fmt.Errorf("workflow must contain at least one step")
	}

	return &workflow, nil
}
