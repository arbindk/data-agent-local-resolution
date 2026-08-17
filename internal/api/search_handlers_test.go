package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"everest.local/data-agent-policy-resolver/internal/config"
	"everest.local/data-agent-policy-resolver/internal/contracts"
	"everest.local/data-agent-policy-resolver/internal/portalstate"
	"everest.local/data-agent-policy-resolver/internal/searchindex"
)

func TestSearchReturnsLocalIndexDocuments(t *testing.T) {
	state := portalstate.New("")
	server := &Server{
		cfg:         config.Config{SearchIndexMode: contracts.SearchIndexModeLocal, SearchIndexName: "everest-test", SearchIndexTimeoutSeconds: 5},
		portalState: state,
	}
	server.searchIndex = searchindex.NewConfiguredIndex(searchindex.FromAppConfig(server.cfg), state)
	server.publishSearchDocuments(contracts.SearchIndexDocument{
		Kind:        contracts.SearchIndexKindObject,
		DataStoreID: "finance-azureblob",
		FileID:      "azureblob:finance:payroll.csv",
		Path:        "payroll.csv",
		Title:       "payroll.csv",
		Summary:     "Confidential payroll object",
		Severity:    "high",
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=payroll&scope=objects", nil)
	rr := httptest.NewRecorder()

	server.search(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Count int         `json:"count"`
		Items []searchHit `json:"items"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Count != 1 {
		t.Fatalf("count = %d, want 1; body=%s", resp.Count, rr.Body.String())
	}
	if resp.Items[0].Title != "payroll.csv" {
		t.Fatalf("title = %q, want payroll.csv", resp.Items[0].Title)
	}
	if resp.Items[0].Screen != "workbench" {
		t.Fatalf("screen = %q, want workbench", resp.Items[0].Screen)
	}
}
