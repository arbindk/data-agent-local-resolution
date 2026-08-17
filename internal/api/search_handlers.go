package api

import (
	"net/http"
	"sort"
	"strings"

	"everest.local/data-agent-policy-resolver/internal/contracts"
	"everest.local/data-agent-policy-resolver/internal/portalstate"
)

type searchHit struct {
	Kind   string `json:"kind"`
	Title  string `json:"title"`
	Sub    string `json:"sub"`
	Screen string `json:"screen"`
	ID     string `json:"id,omitempty"`
	Source string `json:"source,omitempty"`
	Path   string `json:"path,omitempty"`
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	scope := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("scope")))
	if scope == "" {
		scope = "all"
	}

	hits := []searchHit{}
	if s.portalState != nil {
		for _, doc := range s.portalState.ListSearchDocuments(portalstate.SearchDocumentFilter{Query: q, Limit: 100}) {
			if searchScopeIncludesDocument(scope, doc) {
				hits = append(hits, searchHitFromDocument(doc))
			}
		}
		if len(hits) > 0 {
			writeJSON(w, http.StatusOK, map[string]any{
				"query":  q,
				"scope":  scope,
				"source": "index",
				"count":  len(hits),
				"items":  hits,
			})
			return
		}

		hits = searchPortalStateFallback(s.portalState, q, scope)
	}

	if q != "" {
		sort.SliceStable(hits, func(i, j int) bool { return len(hits[i].Title) < len(hits[j].Title) })
	}
	if len(hits) > 100 {
		hits = hits[:100]
	}
	writeJSON(w, http.StatusOK, map[string]any{"query": q, "scope": scope, "source": "portal_state", "count": len(hits), "items": hits})
}

func searchPortalStateFallback(state *portalstate.Store, q string, scope string) []searchHit {
	contains := func(parts ...string) bool {
		if q == "" {
			return true
		}
		for _, p := range parts {
			if strings.Contains(strings.ToLower(p), q) {
				return true
			}
		}
		return false
	}

	hits := []searchHit{}
	opts := state.Options()
	if scope == "all" || scope == "objects" {
		for _, b := range opts.Blobs {
			if contains(b.Name, b.Container) {
				hits = append(hits, searchHit{Kind: contracts.SearchIndexKindObject, Title: b.Name, Sub: strings.TrimSpace(b.Container + " / " + b.AccessTier), Screen: "workbench", ID: b.FileID, Source: b.DataStoreID, Path: b.Name})
			}
		}
	}
	if scope == "all" || scope == "evidence" {
		for _, e := range opts.EvidenceRecords {
			obj := firstNonBlank(e.Blob, e.SubjectID, e.FileID)
			if contains(obj, e.SourceSystem, e.Status, e.Summary) {
				hits = append(hits, searchHit{Kind: contracts.SearchIndexKindEvidence, Title: obj, Sub: strings.TrimSpace(e.Status + " / " + e.EventType), Screen: "evidence", ID: e.EvidenceID, Source: e.DataStoreID, Path: e.Blob})
			}
		}
	}
	if scope == "all" || scope == "decisions" {
		for _, a := range opts.ActionRecords {
			obj := firstNonBlank(a.BlobPath, a.Container, a.FileID)
			if contains(obj, a.Operation, a.Reason, a.GuardianDecision) {
				hits = append(hits, searchHit{Kind: contracts.SearchIndexKindAction, Title: obj, Sub: strings.TrimSpace(a.Operation + " / " + a.Status), Screen: "actions", ID: a.ActionID, Source: a.Container, Path: a.BlobPath})
			}
		}
	}
	if scope == "all" || scope == "audit" {
		for _, ev := range opts.AuditEvents {
			if contains(ev.Summary, ev.EventType) {
				hits = append(hits, searchHit{Kind: contracts.SearchIndexKindAudit, Title: ev.EventType, Sub: ev.Summary, Screen: "diagnostics", ID: ev.EventID})
			}
		}
	}
	return hits
}

func searchScopeIncludesDocument(scope string, doc contracts.SearchIndexDocument) bool {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" || scope == "all" {
		return true
	}
	switch scope {
	case "objects":
		return doc.Kind == contracts.SearchIndexKindObject
	case "findings":
		return doc.Kind == contracts.SearchIndexKindBatchSummary || doc.Kind == contracts.SearchIndexKindDataPassport
	case "evidence":
		return doc.Kind == contracts.SearchIndexKindEvidence
	case "decisions":
		return doc.Kind == contracts.SearchIndexKindDecision || doc.Kind == contracts.SearchIndexKindAction
	case "audit":
		return doc.Kind == contracts.SearchIndexKindAudit
	default:
		return true
	}
}

func searchHitFromDocument(doc contracts.SearchIndexDocument) searchHit {
	title := firstNonBlank(doc.Title, doc.Path, doc.SubjectID, doc.FileID, doc.DocumentID)
	sub := strings.Join(nonEmptyStrings(doc.Container, doc.DataStoreID, doc.Severity, doc.Status), " / ")
	if sub == "" {
		sub = firstNonBlank(doc.Summary, doc.Kind)
	}
	return searchHit{
		Kind:   doc.Kind,
		Title:  title,
		Sub:    sub,
		Screen: searchScreenForDocument(doc),
		ID:     firstNonBlank(doc.FileID, doc.SubjectID, doc.DocumentID),
		Source: doc.DataStoreID,
		Path:   doc.Path,
	}
}

func searchScreenForDocument(doc contracts.SearchIndexDocument) string {
	switch doc.Kind {
	case contracts.SearchIndexKindObject:
		return "workbench"
	case contracts.SearchIndexKindEvidence:
		return "evidence"
	case contracts.SearchIndexKindDecision:
		return "guardian"
	case contracts.SearchIndexKindAction:
		return "actions"
	case contracts.SearchIndexKindDataPassport:
		return "passport"
	case contracts.SearchIndexKindAudit:
		return "diagnostics"
	case contracts.SearchIndexKindBatchSummary:
		return "agent"
	default:
		return "search"
	}
}

func nonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
