package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) NextLease(ctx context.Context, agentID string) (*NextLeaseResponse, error) {
	if strings.TrimSpace(agentID) == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	url := fmt.Sprintf("%s/v1/agents/%s/leases/next", c.baseURL, agentID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get next lease: %w", err)
	}
	defer resp.Body.Close()

	var out NextLeaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode next lease response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("get next lease failed: http %d: %s", resp.StatusCode, out.Message)
	}

	return &out, nil
}

func (c *Client) HeartbeatLease(ctx context.Context, leaseID string, hb LeaseHeartbeatRequest) (*LeaseHeartbeatResponse, error) {
	if strings.TrimSpace(leaseID) == "" {
		return nil, fmt.Errorf("lease_id is required")
	}

	url := fmt.Sprintf("%s/v1/leases/%s/heartbeat", c.baseURL, leaseID)

	body, err := json.Marshal(hb)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("heartbeat lease: %w", err)
	}
	defer resp.Body.Close()

	var out LeaseHeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode heartbeat response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("heartbeat lease failed: http %d", resp.StatusCode)
	}

	return &out, nil
}

func (c *Client) PostJSON(ctx context.Context, path string, payload any) error {
	url := c.baseURL + path

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("post %s failed: http %d", path, resp.StatusCode)
	}

	return nil
}

func (c *Client) UploadExecutionStatus(ctx context.Context, executionID string, status any) error {
	return c.PostJSON(ctx, "/v1/executions/"+executionID+"/status", status)
}

func (c *Client) UploadManifestSummary(ctx context.Context, summary any) error {
	return c.PostJSON(ctx, "/v1/manifests/summary", summary)
}

func (c *Client) UploadManifestDetails(ctx context.Context, executionID string, items any) error {
	return c.PostJSON(ctx, "/v1/manifests/details", map[string]any{
		"execution_id": executionID,
		"items":        items,
	})
}

func (c *Client) UploadProvenance(ctx context.Context, executionID string, items any) error {
	return c.PostJSON(ctx, "/v1/provenance", map[string]any{
		"execution_id": executionID,
		"items":        items,
	})
}

func (c *Client) UploadActionResults(ctx context.Context, executionID string, items any) error {
	return c.PostJSON(ctx, "/v1/actions/results", map[string]any{
		"execution_id": executionID,
		"items":        items,
	})
}

func (c *Client) ListApprovals(ctx context.Context, status string, executionID string) ([]ApprovalRequest, error) {
	values := url.Values{}
	if strings.TrimSpace(status) != "" {
		values.Set("status", status)
	}
	if strings.TrimSpace(executionID) != "" {
		values.Set("execution_id", executionID)
	}

	path := "/v1/approvals"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list approvals: %w", err)
	}
	defer resp.Body.Close()

	var out ApprovalListResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode approvals response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("list approvals failed: http %d", resp.StatusCode)
	}

	return out.Items, nil
}

func (c *Client) CreateApproval(ctx context.Context, payload ApprovalCreateRequest) (*ApprovalRequest, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/approvals", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create approval: %w", err)
	}
	defer resp.Body.Close()

	var out ApprovalCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode create approval response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("create approval failed: http %d", resp.StatusCode)
	}

	return &out.Approval, nil
}
