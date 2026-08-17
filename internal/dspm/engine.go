package dspm

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	TierMetadata = "metadata"
	TierRules    = "rules"
	TierModel    = "model"
	TierDeep     = "deep"
)

type Request struct {
	SourceSystem  string            `json:"source_system,omitempty"`
	ConnectorID   string            `json:"connector_id,omitempty"`
	FileID        string            `json:"file_id,omitempty"`
	Container     string            `json:"container,omitempty"`
	Path          string            `json:"path,omitempty"`
	ContentType   string            `json:"content_type,omitempty"`
	SizeBytes     int64             `json:"size_bytes,omitempty"`
	SampleText    string            `json:"sample_text,omitempty"`
	ScanTier      string            `json:"scan_tier,omitempty"`
	ModelProfile  string            `json:"model_profile,omitempty"`
	EnabledChecks []string          `json:"enabled_checks,omitempty"`
	KeywordProfile string            `json:"keyword_profile,omitempty"`
	KeywordPatterns []KeywordPattern  `json:"keyword_patterns,omitempty"`
	KeywordExclusions []string        `json:"keyword_exclusions,omitempty"`
	Region        string            `json:"region,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
}

type Result struct {
	Status              string            `json:"status"`
	ScanID              string            `json:"scan_id"`
	ScanTier            string            `json:"scan_tier"`
	ModelProfile        string            `json:"model_profile"`
	Classification      string            `json:"classification"`
	Confidence          float64           `json:"confidence"`
	RiskScore           int               `json:"risk_score"`
	Entities            []EntityMatch     `json:"entities"`
	Checks              []CheckResult     `json:"checks"`
	RecommendedMetadata map[string]string `json:"recommended_metadata,omitempty"`
	RecommendedTags     map[string]string `json:"recommended_tags,omitempty"`
	NextTier            string            `json:"next_tier,omitempty"`
	RequiresHITL        bool              `json:"requires_hitl"`
	Summary             string            `json:"summary"`
	CompletedAt         string            `json:"completed_at"`
}

type EntityMatch struct {
	Type          string  `json:"type"`
	ValueRedacted string  `json:"value_redacted"`
	Detector      string  `json:"detector"`
	Confidence    float64 `json:"confidence"`
	Severity      string  `json:"severity,omitempty"`
	Field         string  `json:"field,omitempty"`
	Rationale     string  `json:"rationale,omitempty"`
}

type CheckResult struct {
	Check    string `json:"check"`
	Result   string `json:"result"`
	Severity string `json:"severity"`
	Detail   string `json:"detail"`
}

type ModelProfile struct {
	ID              string   `json:"id"`
	DisplayName     string   `json:"display_name"`
	Runtime         string   `json:"runtime"`
	NetworkMode     string   `json:"network_mode"`
	EntityTypes     []string `json:"entity_types"`
	Description     string   `json:"description"`
	RequiresRuntime bool     `json:"requires_runtime"`
}

type KeywordPattern struct {
	ID            string   `json:"id,omitempty"`
	Pattern       string   `json:"pattern"`
	MatchMode     string   `json:"match_mode,omitempty"`
	Severity      string   `json:"severity,omitempty"`
	EntityType    string   `json:"entity_type,omitempty"`
	CaseSensitive bool     `json:"case_sensitive,omitempty"`
	Scopes        []string `json:"scopes,omitempty"`
	Description   string   `json:"description,omitempty"`
}

type KeywordPack struct {
	ID          string           `json:"id"`
	DisplayName string           `json:"display_name"`
	Description string           `json:"description"`
	Patterns    []KeywordPattern `json:"patterns"`
}

func Scan(req Request) Result {
	now := time.Now().UTC()
	tier := normalizeTier(req.ScanTier)
	modelProfile := strings.TrimSpace(req.ModelProfile)
	if modelProfile == "" {
		modelProfile = "builtin-rules"
	}

	result := Result{
		Status:       "completed",
		ScanID:       scanID(req, now),
		ScanTier:     tier,
		ModelProfile: modelProfile,
		Checks:       make([]CheckResult, 0),
		Entities:     make([]EntityMatch, 0),
		CompletedAt:  now.Format(time.RFC3339),
	}

	result.Checks = append(result.Checks, metadataChecks(req)...)
	keywordEntities := detectKeywordEntities(req, tier)
	result.Entities = append(result.Entities, keywordEntities...)
	result.Checks = append(result.Checks, keywordChecks(keywordEntities)...)
	if tier != TierMetadata && strings.TrimSpace(req.SampleText) != "" {
		result.Entities = append(result.Entities, detectRuleEntities(req.SampleText)...)
		result.Checks = append(result.Checks, entityChecks(result.Entities)...)
	}
	if tier == TierModel || tier == TierDeep {
		result.Checks = append(result.Checks, modelReadinessCheck(modelProfile))
	}

	result.Classification, result.Confidence, result.RiskScore = classify(req, result.Entities, result.Checks)
	result.NextTier = nextTier(tier, result.RiskScore, len(result.Entities), strings.TrimSpace(req.SampleText) != "")
	result.RequiresHITL = result.RiskScore >= 80 || result.Classification == "restricted"
	result.RecommendedMetadata = recommendedMetadata(req, result)
	result.RecommendedTags = recommendedTags(result)
	result.Summary = summary(result)

	return result
}

func SupportedChecks() []CheckResult {
	return []CheckResult{
		{Check: "metadata_sensitive_path", Result: "available", Severity: "medium", Detail: "Flags sensitive path and file-name patterns."},
		{Check: "content_email", Result: "available", Severity: "medium", Detail: "Detects email addresses with built-in rules."},
		{Check: "content_phone", Result: "available", Severity: "medium", Detail: "Detects phone-like identifiers with built-in rules."},
		{Check: "content_credit_card", Result: "available", Severity: "high", Detail: "Detects possible credit cards with Luhn validation."},
		{Check: "content_ssn", Result: "available", Severity: "high", Detail: "Detects US SSN-like identifiers."},
		{Check: "content_india_pan", Result: "available", Severity: "high", Detail: "Detects India PAN-like identifiers."},
		{Check: "content_secret", Result: "available", Severity: "critical", Detail: "Detects API keys, tokens, passwords, and secrets."},
		{Check: "keyword_pattern", Result: "available", Severity: "configurable", Detail: "Detects built-in and customer-defined keyword patterns across path, metadata, tags, and sampled content."},
		{Check: "model_profile_ready", Result: "available", Severity: "info", Detail: "Reports whether a local or external model profile is configured."},
	}
}

func ModelProfiles() []ModelProfile {
	return []ModelProfile{
		{
			ID:              "builtin-rules",
			DisplayName:     "Built-in DSPM Rules",
			Runtime:         "builtin",
			NetworkMode:     "offline",
			EntityTypes:     []string{"EMAIL", "PHONE", "CREDIT_CARD", "SSN", "INDIA_PAN", "SECRET"},
			Description:     "Fast local rules for metadata and sampled content scans.",
			RequiresRuntime: false,
		},
		{
			ID:              "local-bert-ner",
			DisplayName:     "Local BERT NER",
			Runtime:         "onnx_or_local_service",
			NetworkMode:     "air_gapped_supported",
			EntityTypes:     []string{"PERSON", "ORG", "LOCATION", "EMAIL", "PHONE", "CUSTOM_ENTITY"},
			Description:     "Customer-managed BERT/NER profile for private or air-gapped deployments.",
			RequiresRuntime: true,
		},
		{
			ID:              "domain-bert-dspm",
			DisplayName:     "Domain BERT DSPM",
			Runtime:         "customer_plugin",
			NetworkMode:     "private_network",
			EntityTypes:     []string{"CONTRACT_ID", "CUSTOMER_ID", "PROJECT_CODE", "REGULATED_TERM"},
			Description:     "Customer-trained domain model exposed through the connector/model runtime contract.",
			RequiresRuntime: true,
		},
		{
			ID:              "cloud-language-service",
			DisplayName:     "Cloud Language Service",
			Runtime:         "external_api",
			NetworkMode:     "cloud_connected",
			EntityTypes:     []string{"PERSON", "ORG", "LOCATION", "PII", "PHI"},
			Description:     "Cloud AI language provider profile for connected environments.",
			RequiresRuntime: true,
		},
	}
}

func EntityTypes() []string {
	return []string{"EMAIL", "PHONE", "CREDIT_CARD", "SSN", "INDIA_PAN", "SECRET", "KEYWORD", "BUSINESS_SENSITIVE", "LEGAL_TERM", "FINANCIAL_TERM", "PERSON", "ORG", "LOCATION", "CUSTOM_ENTITY"}
}

func KeywordPacks() []KeywordPack {
	return []KeywordPack{
		{
			ID:          "enterprise-sensitive",
			DisplayName: "Enterprise Sensitive Terms",
			Description: "General business-sensitive keywords for board, executive, legal, finance, and confidential material.",
			Patterns: []KeywordPattern{
				{ID: "confidential", Pattern: "confidential", MatchMode: "contains", Severity: "high", EntityType: "BUSINESS_SENSITIVE", Scopes: []string{"path", "metadata", "tags", "content"}},
				{ID: "board", Pattern: "board meeting", MatchMode: "phrase", Severity: "high", EntityType: "BUSINESS_SENSITIVE", Scopes: []string{"path", "metadata", "content"}},
				{ID: "merger", Pattern: "merger", MatchMode: "contains", Severity: "critical", EntityType: "BUSINESS_SENSITIVE", Scopes: []string{"path", "metadata", "content"}},
				{ID: "acquisition", Pattern: "acquisition", MatchMode: "contains", Severity: "critical", EntityType: "BUSINESS_SENSITIVE", Scopes: []string{"path", "metadata", "content"}},
				{ID: "executive", Pattern: "executive", MatchMode: "contains", Severity: "medium", EntityType: "BUSINESS_SENSITIVE", Scopes: []string{"path", "metadata", "content"}},
				{ID: "restricted", Pattern: "restricted", MatchMode: "contains", Severity: "high", EntityType: "BUSINESS_SENSITIVE", Scopes: []string{"path", "metadata", "tags", "content"}},
			},
		},
		{
			ID:          "legal-contracts",
			DisplayName: "Legal and Contract Terms",
			Description: "Terms that often indicate legal agreements, disputes, regulated contract data, or privileged material.",
			Patterns: []KeywordPattern{
				{ID: "nda", Pattern: "NDA", MatchMode: "phrase", Severity: "high", EntityType: "LEGAL_TERM", CaseSensitive: false, Scopes: []string{"path", "metadata", "content"}},
				{ID: "legal_hold", Pattern: "legal hold", MatchMode: "phrase", Severity: "critical", EntityType: "LEGAL_TERM", Scopes: []string{"path", "metadata", "content"}},
				{ID: "privileged", Pattern: "privileged", MatchMode: "contains", Severity: "critical", EntityType: "LEGAL_TERM", Scopes: []string{"path", "metadata", "content"}},
				{ID: "contract", Pattern: "contract", MatchMode: "contains", Severity: "medium", EntityType: "LEGAL_TERM", Scopes: []string{"path", "metadata", "content"}},
				{ID: "litigation", Pattern: "litigation", MatchMode: "contains", Severity: "critical", EntityType: "LEGAL_TERM", Scopes: []string{"path", "metadata", "content"}},
			},
		},
		{
			ID:          "finance-hr",
			DisplayName: "Finance and HR Terms",
			Description: "Payroll, compensation, finance, and workforce-sensitive terms.",
			Patterns: []KeywordPattern{
				{ID: "payroll", Pattern: "payroll", MatchMode: "contains", Severity: "high", EntityType: "FINANCIAL_TERM", Scopes: []string{"path", "metadata", "content"}},
				{ID: "salary", Pattern: "salary", MatchMode: "contains", Severity: "high", EntityType: "FINANCIAL_TERM", Scopes: []string{"path", "metadata", "content"}},
				{ID: "bonus", Pattern: "bonus", MatchMode: "contains", Severity: "medium", EntityType: "FINANCIAL_TERM", Scopes: []string{"path", "metadata", "content"}},
				{ID: "bank_account", Pattern: "bank account", MatchMode: "phrase", Severity: "high", EntityType: "FINANCIAL_TERM", Scopes: []string{"path", "metadata", "content"}},
				{ID: "invoice", Pattern: "invoice", MatchMode: "contains", Severity: "medium", EntityType: "FINANCIAL_TERM", Scopes: []string{"path", "metadata", "content"}},
			},
		},
		{
			ID:          "technology-secrets",
			DisplayName: "Technology and Secret Terms",
			Description: "Engineering, credential, source-code, and project-sensitive keywords.",
			Patterns: []KeywordPattern{
				{ID: "source_code", Pattern: "source code", MatchMode: "phrase", Severity: "high", EntityType: "BUSINESS_SENSITIVE", Scopes: []string{"path", "metadata", "content"}},
				{ID: "private_key", Pattern: "private key", MatchMode: "phrase", Severity: "critical", EntityType: "SECRET", Scopes: []string{"path", "metadata", "content"}},
				{ID: "api_key", Pattern: "api key", MatchMode: "phrase", Severity: "critical", EntityType: "SECRET", Scopes: []string{"path", "metadata", "content"}},
				{ID: "token", Pattern: "token", MatchMode: "contains", Severity: "high", EntityType: "SECRET", Scopes: []string{"path", "metadata", "content"}},
				{ID: "codename", Pattern: "codename", MatchMode: "contains", Severity: "medium", EntityType: "BUSINESS_SENSITIVE", Scopes: []string{"path", "metadata", "content"}},
			},
		},
	}
}

func normalizeTier(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case TierMetadata:
		return TierMetadata
	case TierModel:
		return TierModel
	case TierDeep:
		return TierDeep
	default:
		return TierRules
	}
}

func metadataChecks(req Request) []CheckResult {
	checks := make([]CheckResult, 0)
	path := strings.ToLower(req.Path + " " + req.FileID)
	sensitiveTerms := []string{"payroll", "salary", "ssn", "confidential", "contract", "finance", "medical", "patient", "secret", "token"}
	for _, term := range sensitiveTerms {
		if strings.Contains(path, term) {
			checks = append(checks, CheckResult{
				Check:    "metadata_sensitive_path",
				Result:   "warning",
				Severity: "medium",
				Detail:   "Path or file id contains sensitive term: " + term,
			})
			break
		}
	}
	if req.ContentType == "" {
		checks = append(checks, CheckResult{Check: "metadata_content_type", Result: "warning", Severity: "low", Detail: "Content type is missing."})
	}
	if req.SizeBytes > 100*1024*1024 {
		checks = append(checks, CheckResult{Check: "metadata_large_object", Result: "warning", Severity: "medium", Detail: "Large object should use tiered sampling or delegated scan."})
	}
	return checks
}

func detectKeywordEntities(req Request, tier string) []EntityMatch {
	patterns := keywordPatternsFor(req)
	if len(patterns) == 0 {
		return nil
	}

	sources := map[string]string{
		"path":     strings.TrimSpace(req.Path + " " + req.FileID + " " + req.Container),
		"metadata": joinStringMap(req.Metadata),
		"tags":     joinStringMap(req.Tags),
	}
	if tier != TierMetadata {
		sources["content"] = req.SampleText
	}

	exclusions := req.KeywordExclusions
	seen := map[string]bool{}
	entities := make([]EntityMatch, 0)
	for _, pattern := range patterns {
		pattern.Pattern = strings.TrimSpace(pattern.Pattern)
		if pattern.Pattern == "" {
			continue
		}
		if pattern.EntityType == "" {
			pattern.EntityType = "KEYWORD"
		}
		if pattern.Severity == "" {
			pattern.Severity = "medium"
		}
		scopes := pattern.Scopes
		if len(scopes) == 0 {
			scopes = []string{"path", "metadata", "tags", "content"}
		}

		for _, scope := range scopes {
			text := sources[strings.ToLower(strings.TrimSpace(scope))]
			if strings.TrimSpace(text) == "" || keywordExcluded(text, exclusions) {
				continue
			}
			matches := keywordMatches(text, pattern)
			for _, match := range matches {
				key := pattern.EntityType + ":" + scope + ":" + strings.ToLower(match)
				if seen[key] {
					continue
				}
				seen[key] = true
				entities = append(entities, EntityMatch{
					Type:          pattern.EntityType,
					ValueRedacted: redact(match),
					Detector:      "keyword_" + valueOr(pattern.ID, normalizeID(pattern.Pattern)),
					Confidence:    keywordConfidence(pattern),
					Severity:      normalizeSeverity(pattern.Severity),
					Field:         scope,
					Rationale:     valueOr(pattern.Description, "Matched DSPM keyword pattern: "+pattern.Pattern),
				})
			}
		}
	}

	sort.SliceStable(entities, func(i, j int) bool {
		if entities[i].Severity == entities[j].Severity {
			if entities[i].Type == entities[j].Type {
				return entities[i].ValueRedacted < entities[j].ValueRedacted
			}
			return entities[i].Type < entities[j].Type
		}
		return severityRank(entities[i].Severity) > severityRank(entities[j].Severity)
	})
	return entities
}

func keywordPatternsFor(req Request) []KeywordPattern {
	profile := strings.ToLower(strings.TrimSpace(req.KeywordProfile))
	patterns := make([]KeywordPattern, 0)
	if profile != "none" {
		for _, pack := range KeywordPacks() {
			if profile == "" || profile == "builtin-default" || profile == "all" || profile == pack.ID {
				patterns = append(patterns, pack.Patterns...)
			}
		}
	}
	patterns = append(patterns, req.KeywordPatterns...)
	return patterns
}

func keywordMatches(text string, pattern KeywordPattern) []string {
	mode := strings.ToLower(strings.TrimSpace(pattern.MatchMode))
	if mode == "" {
		mode = "contains"
	}

	searchText := text
	searchPattern := pattern.Pattern
	if !pattern.CaseSensitive && mode != "regex" {
		searchText = strings.ToLower(searchText)
		searchPattern = strings.ToLower(searchPattern)
	}

	switch mode {
	case "regex":
		expr := pattern.Pattern
		if !pattern.CaseSensitive {
			expr = "(?i)" + expr
		}
		re, err := regexp.Compile(expr)
		if err != nil {
			return nil
		}
		return re.FindAllString(text, 25)
	case "word":
		expr := regexp.QuoteMeta(pattern.Pattern)
		if !pattern.CaseSensitive {
			expr = `(?i)\b` + expr + `\b`
		} else {
			expr = `\b` + expr + `\b`
		}
		re, err := regexp.Compile(expr)
		if err != nil {
			return nil
		}
		return re.FindAllString(text, 25)
	default:
		if strings.Contains(searchText, searchPattern) {
			return []string{pattern.Pattern}
		}
		return nil
	}
}

func keywordChecks(entities []EntityMatch) []CheckResult {
	counts := map[string]int{}
	severityByType := map[string]string{}
	for _, entity := range entities {
		if !strings.HasPrefix(entity.Detector, "keyword_") {
			continue
		}
		counts[entity.Type]++
		if severityRank(entity.Severity) > severityRank(severityByType[entity.Type]) {
			severityByType[entity.Type] = entity.Severity
		}
	}
	if len(counts) == 0 {
		return nil
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	checks := make([]CheckResult, 0, len(keys))
	for _, key := range keys {
		severity := normalizeSeverity(severityByType[key])
		checks = append(checks, CheckResult{
			Check:    "keyword_pattern_" + strings.ToLower(key),
			Result:   "warning",
			Severity: severity,
			Detail:   fmt.Sprintf("%d %s keyword matches detected.", counts[key], key),
		})
	}
	return checks
}

func keywordExcluded(text string, exclusions []string) bool {
	lower := strings.ToLower(text)
	for _, exclusion := range exclusions {
		exclusion = strings.TrimSpace(exclusion)
		if exclusion == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(exclusion)) {
			return true
		}
	}
	return false
}

func joinStringMap(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values)*2)
	for key, value := range values {
		parts = append(parts, key, value)
	}
	sort.Strings(parts)
	return strings.Join(parts, " ")
}

func keywordConfidence(pattern KeywordPattern) float64 {
	switch strings.ToLower(strings.TrimSpace(pattern.MatchMode)) {
	case "regex":
		return 0.84
	case "word", "phrase":
		return 0.82
	default:
		return 0.76
	}
}

func detectRuleEntities(text string) []EntityMatch {
	detectors := []struct {
		entityType string
		detector   string
		confidence float64
		pattern    *regexp.Regexp
	}{
		{"EMAIL", "builtin_regex_email", 0.92, regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)},
		{"SSN", "builtin_regex_ssn", 0.88, regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)},
		{"INDIA_PAN", "builtin_regex_india_pan", 0.86, regexp.MustCompile(`\b[A-Z]{5}[0-9]{4}[A-Z]\b`)},
		{"PHONE", "builtin_regex_phone", 0.74, regexp.MustCompile(`\b(?:\+?\d{1,3}[-.\s]?)?(?:\(?\d{3}\)?[-.\s]?)\d{3}[-.\s]?\d{4}\b`)},
	}

	seen := map[string]bool{}
	entities := make([]EntityMatch, 0)
	for _, detector := range detectors {
		for _, value := range detector.pattern.FindAllString(text, 25) {
			key := detector.entityType + ":" + value
			if seen[key] {
				continue
			}
			seen[key] = true
			entities = append(entities, EntityMatch{
				Type:          detector.entityType,
				ValueRedacted: redact(value),
				Detector:      detector.detector,
				Confidence:    detector.confidence,
				Field:         "sample_text",
				Rationale:     "Matched built-in DSPM detector.",
			})
		}
	}

	entities = append(entities, detectCreditCards(text, seen)...)
	entities = append(entities, detectSecrets(text, seen)...)
	sort.SliceStable(entities, func(i, j int) bool {
		if entities[i].Type == entities[j].Type {
			return entities[i].ValueRedacted < entities[j].ValueRedacted
		}
		return entities[i].Type < entities[j].Type
	})
	return entities
}

func detectCreditCards(text string, seen map[string]bool) []EntityMatch {
	pattern := regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`)
	entities := make([]EntityMatch, 0)
	for _, candidate := range pattern.FindAllString(text, 25) {
		digits := onlyDigits(candidate)
		if len(digits) < 13 || len(digits) > 19 || !luhn(digits) {
			continue
		}
		key := "CREDIT_CARD:" + digits
		if seen[key] {
			continue
		}
		seen[key] = true
		entities = append(entities, EntityMatch{
			Type:          "CREDIT_CARD",
			ValueRedacted: redact(digits),
			Detector:      "builtin_luhn_credit_card",
			Confidence:    0.93,
			Field:         "sample_text",
			Rationale:     "Matched card-like number and passed Luhn validation.",
		})
	}
	return entities
}

func detectSecrets(text string, seen map[string]bool) []EntityMatch {
	pattern := regexp.MustCompile(`(?i)\b(api[_-]?key|secret|token|password)\s*[:=]\s*['"]?([A-Za-z0-9_\-./+=]{12,})`)
	entities := make([]EntityMatch, 0)
	for _, match := range pattern.FindAllStringSubmatch(text, 25) {
		if len(match) < 3 {
			continue
		}
		key := "SECRET:" + match[2]
		if seen[key] {
			continue
		}
		seen[key] = true
		entities = append(entities, EntityMatch{
			Type:          "SECRET",
			ValueRedacted: redact(match[2]),
			Detector:      "builtin_regex_secret",
			Confidence:    0.9,
			Field:         "sample_text",
			Rationale:     "Matched token, API key, password, or secret pattern.",
		})
	}
	return entities
}

func entityChecks(entities []EntityMatch) []CheckResult {
	if len(entities) == 0 {
		return []CheckResult{{Check: "content_entities", Result: "ok", Severity: "info", Detail: "No built-in entity detectors matched the supplied sample."}}
	}
	counts := map[string]int{}
	for _, entity := range entities {
		if strings.HasPrefix(entity.Detector, "keyword_") {
			continue
		}
		counts[entity.Type]++
	}
	if len(counts) == 0 {
		return nil
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	checks := make([]CheckResult, 0, len(keys))
	for _, key := range keys {
		severity := "medium"
		if key == "CREDIT_CARD" || key == "SSN" || key == "INDIA_PAN" {
			severity = "high"
		}
		if key == "SECRET" {
			severity = "critical"
		}
		checks = append(checks, CheckResult{
			Check:    "content_" + strings.ToLower(key),
			Result:   "warning",
			Severity: severity,
			Detail:   fmt.Sprintf("%d %s entities detected.", counts[key], key),
		})
	}
	return checks
}

func modelReadinessCheck(profile string) CheckResult {
	for _, p := range ModelProfiles() {
		if p.ID == profile {
			if p.RequiresRuntime {
				return CheckResult{
					Check:    "model_profile_ready",
					Result:   "pending_runtime",
					Severity: "info",
					Detail:   p.DisplayName + " is a supported profile. Connect the model runtime before using it for live inference.",
				}
			}
			return CheckResult{Check: "model_profile_ready", Result: "ok", Severity: "info", Detail: p.DisplayName + " is available locally."}
		}
	}
	return CheckResult{Check: "model_profile_ready", Result: "warning", Severity: "medium", Detail: "Unknown model profile: " + profile}
}

func classify(req Request, entities []EntityMatch, checks []CheckResult) (string, float64, int) {
	risk := 20
	classification := "internal"
	confidence := 0.65

	for _, check := range checks {
		switch check.Severity {
		case "critical":
			risk += 45
		case "high":
			risk += 30
		case "medium":
			risk += 15
		case "low":
			risk += 5
		}
	}
	for _, entity := range entities {
		switch normalizeSeverity(entity.Severity) {
		case "critical":
			risk += 35
			classification = "restricted"
			confidence = maxFloat(confidence, entity.Confidence)
		case "high":
			risk += 22
			if classification != "restricted" {
				classification = "confidential"
			}
			confidence = maxFloat(confidence, entity.Confidence)
		case "medium":
			risk += 10
			if classification == "internal" {
				classification = "confidential"
			}
			confidence = maxFloat(confidence, entity.Confidence)
		}
		switch entity.Type {
		case "SECRET":
			risk += 35
			classification = "restricted"
			confidence = maxFloat(confidence, entity.Confidence)
		case "CREDIT_CARD", "SSN", "INDIA_PAN":
			risk += 25
			if classification != "restricted" {
				classification = "confidential"
			}
			confidence = maxFloat(confidence, entity.Confidence)
		case "EMAIL", "PHONE":
			risk += 8
			if classification == "internal" {
				classification = "confidential"
			}
			confidence = maxFloat(confidence, entity.Confidence)
		case "BUSINESS_SENSITIVE", "LEGAL_TERM", "FINANCIAL_TERM", "KEYWORD":
			if classification == "internal" {
				classification = "confidential"
			}
		}
	}
	if strings.Contains(strings.ToLower(req.Path), "public") && len(entities) == 0 {
		classification = "public"
		confidence = 0.6
	}
	if risk > 100 {
		risk = 100
	}
	return classification, confidence, risk
}

func nextTier(tier string, risk int, entityCount int, hasSample bool) string {
	switch tier {
	case TierMetadata:
		if risk >= 35 || !hasSample {
			return TierRules
		}
	case TierRules:
		if risk >= 60 || entityCount > 0 {
			return TierModel
		}
	case TierModel:
		if risk >= 80 {
			return TierDeep
		}
	}
	return ""
}

func recommendedMetadata(req Request, result Result) map[string]string {
	return map[string]string{
		"everest_dspm_reviewed_at": result.CompletedAt,
		"everest_dspm_tier":        result.ScanTier,
		"everest_dspm_model":       result.ModelProfile,
		"everest_dspm_summary":     result.Summary,
		"everest_content_type":     valueOr(req.ContentType, "unknown"),
	}
}

func recommendedTags(result Result) map[string]string {
	types := make([]string, 0, len(result.Entities))
	seen := map[string]bool{}
	for _, entity := range result.Entities {
		if !seen[entity.Type] {
			seen[entity.Type] = true
			types = append(types, strings.ToLower(entity.Type))
		}
	}
	sort.Strings(types)
	if len(types) == 0 {
		types = append(types, "none")
	}
	return map[string]string{
		"everest_dspm_classification": result.Classification,
		"everest_dspm_risk":           strconv.Itoa(result.RiskScore),
		"everest_dspm_entities":       strings.Join(types, "_"),
		"everest_dspm_hitl":           strconv.FormatBool(result.RequiresHITL),
	}
}

func summary(result Result) string {
	if len(result.Entities) == 0 {
		return fmt.Sprintf("%s scan found no built-in entity matches; risk score %d.", result.ScanTier, result.RiskScore)
	}
	counts := map[string]int{}
	for _, entity := range result.Entities {
		counts[entity.Type]++
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return fmt.Sprintf("%s scan detected %s; risk score %d.", result.ScanTier, strings.Join(parts, ", "), result.RiskScore)
}

func scanID(req Request, now time.Time) string {
	h := sha1.New()
	_, _ = h.Write([]byte(req.FileID))
	_, _ = h.Write([]byte(req.Path))
	_, _ = h.Write([]byte(now.Format(time.RFC3339Nano)))
	return "dspm-" + hex.EncodeToString(h.Sum(nil))[:16]
}

func redact(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	if strings.Contains(value, "@") {
		parts := strings.SplitN(value, "@", 2)
		return firstRunes(parts[0], 2) + "***@" + parts[1]
	}
	return firstRunes(value, 2) + "***" + lastRunes(value, 2)
}

func onlyDigits(value string) string {
	var out strings.Builder
	for _, r := range value {
		if unicode.IsDigit(r) {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func luhn(digits string) bool {
	sum := 0
	double := false
	for i := len(digits) - 1; i >= 0; i-- {
		n := int(digits[i] - '0')
		if double {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		double = !double
	}
	return sum%10 == 0
}

func firstRunes(value string, n int) string {
	runes := []rune(value)
	if len(runes) < n {
		return value
	}
	return string(runes[:n])
}

func lastRunes(value string, n int) string {
	runes := []rune(value)
	if len(runes) < n {
		return value
	}
	return string(runes[len(runes)-n:])
}

func valueOr(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func normalizeSeverity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "low":
		return "low"
	case "info":
		return "info"
	default:
		return "medium"
	}
}

func severityRank(value string) int {
	switch normalizeSeverity(value) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func normalizeID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteRune('_')
			lastDash = true
		}
	}
	return strings.Trim(out.String(), "_")
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func Extension(path string) string {
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
}
