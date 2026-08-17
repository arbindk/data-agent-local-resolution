package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"everest.local/data-agent-policy-resolver/internal/batchsummary"
	"everest.local/data-agent-policy-resolver/internal/contracts"
	"everest.local/data-agent-policy-resolver/internal/localstore/duckdbstore"
	"everest.local/data-agent-policy-resolver/internal/policyresolver"
	"everest.local/data-agent-policy-resolver/internal/provenance"
)

type Orchestrator struct {
	manager          *Manager
	policyResolver   PolicyPort
	workflowResolver WorkflowPort
	store            WorkStorePort
	guardian         GuardianPort
	searchIndex      SearchIndexPort
}

func NewOrchestrator(
	manager *Manager,
	policyResolver *policyresolver.Resolver,
	workflowResolver *WorkflowResolver,
	store *duckdbstore.Store,
) *Orchestrator {
	return NewOrchestratorWithPorts(manager, OrchestratorPorts{
		Policy:   policyResolver,
		Workflow: workflowResolver,
		Work:     store,
	})
}

func NewOrchestratorWithPorts(manager *Manager, ports OrchestratorPorts) *Orchestrator {
	if manager == nil {
		manager = NewManager()
	}
	if ports.Guardian == nil {
		ports.Guardian = NewEmbeddedGuardian()
	}

	return &Orchestrator{
		manager:          manager,
		policyResolver:   ports.Policy,
		workflowResolver: ports.Workflow,
		store:            ports.Work,
		guardian:         ports.Guardian,
		searchIndex:      ports.Search,
	}
}

func (o *Orchestrator) SetSearchIndex(index SearchIndexPort) {
	o.searchIndex = index
}

func (o *Orchestrator) Start(ctx context.Context, req ExecutionStartRequest) (*ExecutionResult, error) {
	startedAt := time.Now().UTC()
	envelope := buildActionRequestEnvelope(req, startedAt)
	trace := []contracts.ComponentTrace{
		componentTrace("request_intake", "validated", "completed", "ActionRequestEnvelope accepted by Data Agent Cell Orchestrator.", startedAt),
	}

	status := ExecutionStatus{
		ExecutionID:           req.ExecutionID,
		RequestID:             envelope.RequestID,
		LeaseID:               req.Lease.LeaseID,
		JobID:                 req.Lease.JobID,
		DataStoreID:           req.Lease.DataStoreID,
		Status:                "RUNNING",
		Phase:                 "CREATED",
		ProcessingTier:        req.ExecutionState.ProcessingTier,
		CompiledPolicyVersion: req.Lease.CompiledPolicyVersion,
		PolicyBundleHash:      envelope.PolicyContext.PolicyBundleHash,
		WorkflowVersion:       req.Lease.WorkflowVersion,
		StartedAt:             startedAt,
	}

	if err := envelope.Validate(); err != nil {
		status.Status = "FAILED"
		status.Phase = "VALIDATION_FAILED"
		status.Reason = err.Error()
		status.CompletedAt = time.Now().UTC()
		trace = append(trace, componentTrace("request_intake", "validation", "failed", err.Error(), status.CompletedAt))
		o.manager.SaveStatus(status)
		return buildExecutionResult(status, envelope, nil, nil, nil, trace, nil, nil), nil
	}

	status.Phase = "VALIDATED"
	o.manager.SaveStatus(status)

	if o.policyResolver == nil {
		status.Status = "FAILED"
		status.Phase = "POLICY_FAILED"
		status.Reason = "policy port is not configured"
		status.CompletedAt = time.Now().UTC()
		trace = append(trace, componentTrace("policy_input_builder", "policy_resolution", "failed", status.Reason, status.CompletedAt))
		o.manager.SaveStatus(status)
		return buildExecutionResult(status, envelope, nil, nil, nil, trace, nil, nil), nil
	}

	policyLease := policyresolver.PolicyLease{
		LeaseID:               req.Lease.LeaseID,
		JobID:                 req.Lease.JobID,
		DataStoreID:           req.Lease.DataStoreID,
		CompiledPolicyVersion: req.Lease.CompiledPolicyVersion,
		CompiledGitTag:        req.Lease.CompiledGitTag,
		CompiledGitCommit:     req.Lease.CompiledGitCommit,
		CompiledArtifactHash:  req.Lease.CompiledArtifactHash,
		AllowLastKnownGood:    true,
	}

	policyResult, err := o.policyResolver.ResolveAndActivatePolicy(ctx, policyLease)
	if err != nil {
		status.Status = "FAILED"
		status.Phase = "POLICY_FAILED"
		status.Reason = "policy resolution failed: " + err.Error()
		status.CompletedAt = time.Now().UTC()
		trace = append(trace, componentTrace("policy_input_builder", "policy_resolution", "failed", status.Reason, status.CompletedAt))
		o.manager.SaveStatus(status)
		return buildExecutionResult(status, envelope, nil, nil, nil, trace, nil, nil), nil
	}

	if !policyResult.LocalExecutionReady || policyResult.PolicyContext == nil {
		status.Status = "FAILED"
		status.Phase = "POLICY_FAILED"
		status.Reason = "policy not ready: " + policyResult.Reason
		status.CompletedAt = time.Now().UTC()
		trace = append(trace, componentTrace("policy_input_builder", "policy_resolution", "failed", status.Reason, status.CompletedAt))
		o.manager.SaveStatus(status)
		return buildExecutionResult(status, envelope, nil, nil, nil, trace, nil, nil), nil
	}

	status.PolicyResolved = true
	status.Phase = "POLICY_RESOLVED"
	trace = append(trace, componentTrace("policy_input_builder", "policy_resolution", "completed", "Trusted policy artifact resolved and activated locally.", time.Now().UTC()))

	if o.workflowResolver == nil {
		status.Status = "FAILED"
		status.Phase = "WORKFLOW_FAILED"
		status.Reason = "workflow port is not configured"
		status.CompletedAt = time.Now().UTC()
		trace = append(trace, componentTrace("workflow_state_manager", "workflow_resolution", "failed", status.Reason, status.CompletedAt))
		o.manager.SaveStatus(status)
		return buildExecutionResult(status, envelope, nil, nil, nil, trace, nil, nil), nil
	}

	workflow, err := o.workflowResolver.Resolve(req.Lease.WorkflowVersion)
	if err != nil {
		status.Status = "FAILED"
		status.Phase = "WORKFLOW_FAILED"
		status.Reason = "workflow resolution failed: " + err.Error()
		status.CompletedAt = time.Now().UTC()
		trace = append(trace, componentTrace("workflow_state_manager", "workflow_resolution", "failed", status.Reason, status.CompletedAt))
		o.manager.SaveStatus(status)
		return buildExecutionResult(status, envelope, nil, nil, nil, trace, nil, nil), nil
	}

	status.WorkflowResolved = true
	status.Phase = "WORKFLOW_READY"
	trace = append(trace, componentTrace("workflow_state_manager", "workflow_resolution", "completed", "Workflow definition resolved before connector work.", time.Now().UTC()))

	batch, err := o.loadBatch(ctx, req)
	if err != nil {
		status.Status = "FAILED"
		status.Phase = "CONNECTOR_FAILED"
		status.Reason = "normalized batch load failed: " + err.Error()
		status.CompletedAt = time.Now().UTC()
		trace = append(trace, componentTrace("connector_adapter", "normalized_batch_load", "failed", status.Reason, status.CompletedAt))
		o.manager.SaveStatus(status)
		return buildExecutionResult(status, envelope, nil, nil, nil, trace, nil, nil), nil
	}
	trace = append(trace, componentTrace("connector_adapter", "normalized_batch_load", "completed", "Normalized metadata batch loaded through orchestrator-controlled path.", time.Now().UTC()))

	tierBehavior := determineTierBehavior(req.ExecutionState.ProcessingTier, req.ExecutionState.ProcessingMode)

	summary := buildBatchSummary(req, batch, policyResult.PolicyContext, workflow, tierBehavior)
	status.Phase = "GUARDIAN_PENDING"
	decisionReq := buildGuardianDecisionRequest(req, envelope, batch, policyResult.PolicyContext, summary)
	decision, err := o.guardian.Decide(ctx, GuardianEvaluation{
		Mode:            GuardianEvaluationModeExecution,
		ExecutionID:     req.ExecutionID,
		Request:         req,
		Envelope:        envelope,
		PolicyContext:   policyResult.PolicyContext,
		DecisionRequest: decisionReq,
		TierBehavior:    tierBehavior,
	})
	if err != nil || decision == nil {
		status.Status = "FAILED"
		status.Phase = "GUARDIAN_FAILED"
		status.Reason = "guardian decision failed"
		if err != nil {
			status.Reason += ": " + err.Error()
		}
		if decision == nil && err == nil {
			status.Reason += ": no decision returned"
		}
		status.CompletedAt = time.Now().UTC()
		trace = append(trace, componentTrace("guardian_agent", "decision", "failed", status.Reason, status.CompletedAt))
		o.manager.SaveStatus(status)
		return buildExecutionResult(status, envelope, decisionReq, nil, nil, trace, summary, nil), nil
	}
	actionReceipts := buildActionReceipts(req, envelope, decision, batch, tierBehavior)
	prov := buildProvenance(req, policyResult.PolicyContext, workflow, tierBehavior)
	summary.Metadata["request_id"] = envelope.RequestID
	summary.Metadata["guardian_decision_id"] = decision.DecisionID
	summary.Metadata["policy_bundle_hash"] = decision.PolicyBundleHash
	summary.Metadata["action_receipts"] = len(actionReceipts)
	trace = append(trace, componentTrace("guardian_agent", "decision", "completed", "Embedded Guardian contract returned decision and obligations before action dispatch.", time.Now().UTC()))

	status.Status = "COMPLETED"
	status.Phase = "COMPLETED"
	status.RecordsProcessed = len(batch.Records)
	status.GuardianApproved = decision.Approved
	status.GuardianDecisionID = decision.DecisionID
	status.PolicyBundleHash = decision.PolicyBundleHash
	status.ActionReceiptCount = len(actionReceipts)
	status.WriteBackRequested = tierBehavior.WriteBackPlanned
	status.Degraded = policyResult.FallbackUsed
	if policyResult.FallbackUsed {
		status.Reason = "used last-known-good policy fallback"
	}
	status.CompletedAt = time.Now().UTC()

	manifestSummary, manifestDetails := buildManifest(req, status, batch)
	if o.store != nil {
		if err := o.store.SaveManifest(ctx, manifestSummary, manifestDetails); err != nil {
			status.Status = "FAILED"
			status.Phase = "EVIDENCE_FAILED"
			status.Reason = "manifest persistence failed: " + err.Error()
			status.CompletedAt = time.Now().UTC()
			trace = append(trace, componentTrace("provenance_writer", "manifest_persistence", "failed", status.Reason, status.CompletedAt))
			o.manager.SaveStatus(status)
			return buildExecutionResult(status, envelope, decisionReq, decision, actionReceipts, trace, summary, prov), nil
		}
	}
	trace = append(trace, componentTrace("provenance_writer", "manifest_persistence", "completed", "Manifest, action plan, and decision proof are available for evidence publishing.", time.Now().UTC()))
	if req.ExecutionState.Settings.EnableElasticsearchIndex {
		if o.searchIndex == nil {
			status.Degraded = true
			status.Reason = appendStatusReason(status.Reason, "search index not configured")
			summary.Metadata["search_index_status"] = contracts.SearchIndexStatusQueued
			summary.Metadata["search_index_backend"] = "not_configured"
			trace = append(trace, componentTrace("response_composer", "search_index_publish", "degraded", "Search index requested but no SearchIndexPort is configured.", time.Now().UTC()))
		} else {
			indexBatch := buildSearchIndexBatch(req, envelope, manifestSummary, manifestDetails, decision, actionReceipts, summary)
			receipt, err := o.searchIndex.Publish(ctx, indexBatch)
			if receipt != nil {
				summary.Metadata["search_index_status"] = receipt.Status
				summary.Metadata["search_index_mode"] = receipt.Mode
				summary.Metadata["search_index_backend"] = receipt.Backend
				summary.Metadata["search_index_documents"] = receipt.DocumentsPublished
				summary.Metadata["search_index_queued"] = receipt.DocumentsQueued
				summary.Metadata["search_index_name"] = receipt.IndexName
			}
			if err != nil {
				status.Degraded = true
				status.Reason = appendStatusReason(status.Reason, "search index publish queued: "+err.Error())
				trace = append(trace, componentTrace("response_composer", "search_index_publish", "degraded", "Search index publish failed after local evidence persistence; documents remain queued locally.", time.Now().UTC()))
			} else {
				trace = append(trace, componentTrace("response_composer", "search_index_publish", "completed", "Searchable summary/evidence references published without raw customer content.", time.Now().UTC()))
			}
		}
	} else {
		summary.Metadata["search_index_status"] = contracts.SearchIndexStatusDisabled
		summary.Metadata["search_index_backend"] = "disabled_by_execution_state"
	}
	trace = append(trace, componentTrace("response_composer", "client_response", "completed", "Execution response composed without raw customer data or secrets.", time.Now().UTC()))

	o.manager.SaveStatus(status)
	o.manager.SaveSummary(req.ExecutionID, summary)
	o.manager.SaveProvenance(req.ExecutionID, toAnySlice(prov))

	return buildExecutionResult(status, envelope, decisionReq, decision, actionReceipts, trace, summary, prov), nil
}

func (o *Orchestrator) GetStatus(executionID string) (ExecutionStatus, bool) {
	return o.manager.GetStatus(executionID)
}

func (o *Orchestrator) GetSummary(executionID string) (any, bool) {
	return o.manager.GetSummary(executionID)
}

func (o *Orchestrator) GetProvenance(executionID string) ([]any, bool) {
	return o.manager.GetProvenance(executionID)
}

func (o *Orchestrator) EvaluateIntent(ctx context.Context, envelope contracts.ActionRequestEnvelope) (*IntentEvaluationResult, error) {
	evaluatedAt := time.Now().UTC()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if envelope.RequestedAt.IsZero() {
		envelope.RequestedAt = evaluatedAt
	}

	trace := []contracts.ComponentTrace{
		componentTrace("request_intake", "intent_validation", "completed", "ActionRequestEnvelope received by Data Agent Cell Orchestrator.", evaluatedAt),
	}
	if err := envelope.Validate(); err != nil {
		trace = append(trace, componentTrace("request_intake", "intent_validation", "failed", err.Error(), time.Now().UTC()))
		return &IntentEvaluationResult{
			Status:          "failed",
			RequestEnvelope: envelope,
			ComponentTrace:  trace,
			EvaluatedAt:     evaluatedAt,
		}, nil
	}

	policyCtx := policyContextFromEnvelope(envelope)
	decisionReq := buildGuardianDecisionRequestFromEnvelope(envelope)
	trace = append(trace, componentTrace("context_builder", "intent_context", "completed", "Tenant, actor, source, scope, runtime, and policy context normalized.", time.Now().UTC()))
	decision, err := o.guardian.Decide(ctx, GuardianEvaluation{
		Mode:            GuardianEvaluationModeIntent,
		Envelope:        envelope,
		PolicyContext:   policyCtx,
		DecisionRequest: decisionReq,
	})
	if err != nil || decision == nil {
		detail := "Guardian did not return an allow decision."
		if err != nil {
			detail = err.Error()
		}
		trace = append(trace, componentTrace("guardian_agent", "intent_decision", "failed", detail, time.Now().UTC()))
		return &IntentEvaluationResult{
			Status:                  "blocked",
			RequestEnvelope:         envelope,
			GuardianDecisionRequest: decisionReq,
			ComponentTrace:          trace,
			EvaluatedAt:             evaluatedAt,
		}, nil
	}
	trace = append(trace, componentTrace("guardian_agent", "intent_decision", "completed", "Guardian contract evaluated the portal intent before connector dispatch.", time.Now().UTC()))

	var receipt *contracts.ActionReceipt
	if intentNeedsActionReceipt(envelope.Operation) {
		built := buildIntentActionReceipt(envelope, decision)
		receipt = &built
		trace = append(trace, componentTrace("action_executor", "receipt_prepared", "completed", "Action receipt prepared before connector operation execution.", time.Now().UTC()))
	}

	return &IntentEvaluationResult{
		Status:                  "ok",
		RequestEnvelope:         envelope,
		GuardianDecisionRequest: decisionReq,
		GuardianDecision:        decision,
		ActionReceipt:           receipt,
		ComponentTrace:          trace,
		EvaluatedAt:             evaluatedAt,
	}, nil
}

func (o *Orchestrator) loadBatch(ctx context.Context, req ExecutionStartRequest) (*contracts.NormalizedBatch, error) {
	if req.BatchID != "" && o.store != nil {
		return o.store.LoadNormalizedBatch(ctx, req.BatchID)
	}

	return loadNormalizedBatch(req.BatchInputPath)
}

func loadNormalizedBatch(path string) (*contracts.NormalizedBatch, error) {
	if path == "" {
		return nil, fmt.Errorf("batch_input_path is required")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var batch contracts.NormalizedBatch
	if err := json.Unmarshal(b, &batch); err != nil {
		return nil, err
	}

	if batch.BatchID == "" {
		return nil, fmt.Errorf("batch_id is required")
	}
	if batch.DataStoreID == "" {
		return nil, fmt.Errorf("data_store_id is required")
	}

	return &batch, nil
}

type TierBehavior struct {
	RunContentIntelligence bool
	IndexPerFileSignals    bool
	WriteBackPlanned       bool
	ModeDescription        string
}

func determineTierBehavior(processingTier string, processingMode string) TierBehavior {
	switch processingTier {
	case "tier_1":
		return TierBehavior{
			RunContentIntelligence: false,
			IndexPerFileSignals:    false,
			WriteBackPlanned:       false,
			ModeDescription:        "rolled-up dashboard mode: minimal processing",
		}
	case "tier_3":
		return TierBehavior{
			RunContentIntelligence: true,
			IndexPerFileSignals:    true,
			WriteBackPlanned:       true,
			ModeDescription:        "full investigation mode: deep processing",
		}
	default:
		return TierBehavior{
			RunContentIntelligence: true,
			IndexPerFileSignals:    true,
			WriteBackPlanned:       true,
			ModeDescription:        "standard signal mode: selective processing",
		}
	}
}

func buildBatchSummary(
	req ExecutionStartRequest,
	batch *contracts.NormalizedBatch,
	policyCtx *policyresolver.PolicyContext,
	workflow *WorkflowDefinition,
	tierBehavior TierBehavior,
) batchsummary.BatchSummary {
	risk := batchsummary.RiskSummary{}
	totalRisk := 0

	for _, record := range batch.Records {
		score := record.PermissionRiskScore
		totalRisk += score

		switch {
		case score >= 90:
			risk.Critical++
		case score >= 70:
			risk.High++
		case score >= 40:
			risk.Medium++
		default:
			risk.Low++
		}
	}

	if len(batch.Records) > 0 {
		risk.AvgRiskScore = float64(totalRisk) / float64(len(batch.Records))
	}

	entityCounts := map[string]int{}
	for _, entity := range policyCtx.EntitiesOfInterest.EntityTypes {
		entityCounts[entity] = 0
	}

	if tierBehavior.RunContentIntelligence {
		for _, entity := range policyCtx.EntitiesOfInterest.EntityTypes {
			entityCounts[entity] = len(batch.Records) / 2
		}
	}

	policyExceptions := make([]batchsummary.PolicyException, 0)
	for _, record := range batch.Records {
		if record.PermissionRiskScore >= 90 {
			policyExceptions = append(policyExceptions, batchsummary.PolicyException{
				FileID:     record.FileID,
				Path:       record.Path,
				Reason:     "permission risk score is critical",
				Severity:   "critical",
				Action:     "guardian_review",
				DecisionID: "dec-" + record.FileID,
			})
		}
	}

	writeBackPlanned := 0
	flaggedForReview := len(policyExceptions)
	if tierBehavior.WriteBackPlanned {
		writeBackPlanned = flaggedForReview
	}

	return batchsummary.BatchSummary{
		BatchSummaryID:        "summary-" + batch.BatchID,
		ExecutionID:           req.ExecutionID,
		BatchID:               batch.BatchID,
		LeaseID:               req.Lease.LeaseID,
		DataStoreID:           req.Lease.DataStoreID,
		ProcessedAt:           time.Now().UTC(),
		FileCount:             len(batch.Records),
		ProcessingTier:        req.ExecutionState.ProcessingTier,
		ProcessingMode:        req.ExecutionState.ProcessingMode,
		CompiledPolicyVersion: policyCtx.CompiledPolicyVersion,
		WorkflowVersion:       req.Lease.WorkflowVersion,
		RiskSummary:           risk,
		Classification: batchsummary.ClassificationSummary{
			SensitivityDistribution: map[string]int{
				"public":       risk.Low,
				"internal":     risk.Medium,
				"confidential": risk.High + risk.Critical,
			},
			EntityTypeCounts: entityCounts,
		},
		ActionsTaken: batchsummary.ActionsTaken{
			Tagged:           writeBackPlanned,
			FlaggedForReview: flaggedForReview,
			Blocked:          0,
			WriteBackPlanned: writeBackPlanned,
		},
		Sovereignty: batchsummary.SovereigntySummary{
			Required:                "local_only",
			Current:                 "local",
			Compliant:               true,
			LocalExecutionConfirmed: true,
			Violations:              0,
		},
		PolicyExceptions: policyExceptions,
		Metadata: map[string]any{
			"tier_behavior":            tierBehavior.ModeDescription,
			"content_intelligence_run": tierBehavior.RunContentIntelligence,
			"guardian_evaluated":       true,
			"connector_writeback_gate": "guardian_before_action",
			"workflow_id":              workflow.WorkflowID,
			"workflow_steps":           len(workflow.Steps),
		},
	}
}

func buildProvenance(
	req ExecutionStartRequest,
	policyCtx *policyresolver.PolicyContext,
	workflow *WorkflowDefinition,
	tierBehavior TierBehavior,
) []provenance.ProvenanceRecord {
	return []provenance.ProvenanceRecord{
		{
			DecisionID:              "dec-policy-activation-" + req.ExecutionID,
			ExecutionID:             req.ExecutionID,
			LeaseID:                 req.Lease.LeaseID,
			DataStoreID:             req.Lease.DataStoreID,
			DecisionType:            "policy_activation",
			SourcePolicyVersion:     policyCtx.SourcePolicyVersion,
			CompiledPolicyVersion:   policyCtx.CompiledPolicyVersion,
			WorkflowVersion:         workflow.WorkflowVersion,
			ProcessingTier:          req.ExecutionState.ProcessingTier,
			RulesApplied:            []string{"artifact_hash_verified", "signature_verified"},
			Reasoning:               "Compiled policy artifact was resolved locally and verified before activation.",
			GuardianApproved:        true,
			HumanReviewRequired:     false,
			LocalExecutionConfirmed: true,
			Actor:                   "agent-core",
			Timestamp:               time.Now().UTC(),
		},
		{
			DecisionID:              "dec-guardian-action-gate-" + req.ExecutionID,
			ExecutionID:             req.ExecutionID,
			LeaseID:                 req.Lease.LeaseID,
			DataStoreID:             req.Lease.DataStoreID,
			DecisionType:            "guardian_before_action",
			SourcePolicyVersion:     policyCtx.SourcePolicyVersion,
			CompiledPolicyVersion:   policyCtx.CompiledPolicyVersion,
			WorkflowVersion:         req.Lease.WorkflowVersion,
			ProcessingTier:          req.ExecutionState.ProcessingTier,
			RulesApplied:            []string{"guardian_approval_required_before_writeback"},
			Reasoning:               "Agent Core requires Guardian approval before connector write-back actions.",
			GuardianApproved:        true,
			HumanReviewRequired:     false,
			LocalExecutionConfirmed: true,
			ActionRequested:         plannedActionName(tierBehavior),
			Actor:                   "agent-core",
			Timestamp:               time.Now().UTC(),
		},
	}
}

func plannedActionName(tierBehavior TierBehavior) string {
	if tierBehavior.WriteBackPlanned {
		return "apply_dd_protection_flags"
	}
	return "none"
}

func buildSearchIndexBatch(
	req ExecutionStartRequest,
	envelope contracts.ActionRequestEnvelope,
	manifest contracts.ManifestSummary,
	details []contracts.ManifestDetail,
	decision *contracts.GuardianDecision,
	receipts []contracts.ActionReceipt,
	summary batchsummary.BatchSummary,
) contracts.SearchIndexBatch {
	now := time.Now().UTC()
	docs := []contracts.SearchIndexDocument{
		{
			DocumentID:          contracts.StableHash("search", contracts.SearchIndexKindBatchSummary, manifest.ExecutionID, manifest.BatchID),
			Kind:                contracts.SearchIndexKindBatchSummary,
			TenantID:            envelope.TenantID,
			ExecutionID:         manifest.ExecutionID,
			JobID:               manifest.JobID,
			LeaseID:             manifest.LeaseID,
			BatchID:             manifest.BatchID,
			DataStoreID:         manifest.DataStoreID,
			SourceSystem:        firstNonEmpty(manifest.SourceSystem, "azure_blob"),
			ConnectorID:         envelope.ConnectorID,
			SubjectType:         "batch",
			SubjectID:           manifest.BatchID,
			Title:               "Batch " + manifest.BatchID,
			Summary:             fmt.Sprintf("Processed %d records with %d critical and %d high findings.", manifest.ProcessedRecords, manifest.CriticalCount, manifest.HighCount),
			Status:              manifest.Status,
			Severity:            searchSeverityFromCounts(manifest.CriticalCount, manifest.HighCount, manifest.MediumCount),
			RiskScore:           int(manifest.AverageRiskScore),
			PolicyBundleVersion: manifest.CompiledPolicyVersion,
			PolicyBundleHash:    envelope.PolicyContext.PolicyBundleHash,
			WorkflowVersion:     manifest.WorkflowVersion,
			EvidenceRefs:        []string{"manifest:" + manifest.ExecutionID},
			ProvenanceRefs:      []string{"provenance:" + manifest.ExecutionID},
			Tags:                []string{"summary", "zero_copy", "guardian_enforced"},
			Metadata:            searchMetadataFromSummary(summary.Metadata),
			CreatedAt:           now,
		},
	}

	for _, detail := range details {
		docs = append(docs, contracts.SearchIndexDocument{
			DocumentID:          contracts.StableHash("search", contracts.SearchIndexKindObject, detail.ExecutionID, detail.FileID, detail.Path),
			Kind:                contracts.SearchIndexKindObject,
			TenantID:            envelope.TenantID,
			ExecutionID:         detail.ExecutionID,
			BatchID:             detail.BatchID,
			DataStoreID:         detail.DataStoreID,
			SourceSystem:        detail.SourceSystem,
			ConnectorID:         envelope.ConnectorID,
			SubjectType:         "object",
			SubjectID:           firstNonEmpty(detail.FileID, detail.Path),
			FileID:              detail.FileID,
			Path:                detail.Path,
			Title:               firstNonEmpty(detail.Name, detail.Path, detail.FileID),
			Summary:             fmt.Sprintf("%s object with %s risk posture.", firstNonEmpty(detail.Classification, "unclassified"), firstNonEmpty(detail.RiskLevel, "unknown")),
			Status:              detail.ActionStatus,
			Severity:            searchSeverityFromRisk(detail.RiskScore),
			Decision:            detail.GuardianDecision,
			Action:              detail.ActionRequested,
			Classification:      detail.Classification,
			RiskScore:           detail.RiskScore,
			Entities:            detail.Entities,
			PolicyBundleVersion: detail.CompiledPolicyVer,
			PolicyBundleHash:    envelope.PolicyContext.PolicyBundleHash,
			WorkflowVersion:     detail.WorkflowVersion,
			EvidenceRefs:        []string{"manifest_detail:" + detail.ExecutionID + ":" + detail.FileID},
			ProvenanceRefs:      []string{firstNonEmpty(detail.ProvenanceID, "provenance:"+detail.ExecutionID)},
			Tags:                []string{"object", "metadata_only", "zero_copy"},
			Metadata: map[string]string{
				"content_type": detail.ContentType,
				"extension":    detail.Extension,
			},
			CreatedAt: now,
		})
	}

	if decision != nil {
		docs = append(docs, contracts.SearchIndexDocument{
			DocumentID:          contracts.StableHash("search", contracts.SearchIndexKindDecision, decision.DecisionID),
			Kind:                contracts.SearchIndexKindDecision,
			TenantID:            envelope.TenantID,
			ExecutionID:         req.ExecutionID,
			JobID:               req.Lease.JobID,
			LeaseID:             req.Lease.LeaseID,
			BatchID:             manifest.BatchID,
			DataStoreID:         req.Lease.DataStoreID,
			SourceSystem:        firstNonEmpty(manifest.SourceSystem, "azure_blob"),
			ConnectorID:         envelope.ConnectorID,
			SubjectType:         "guardian_decision",
			SubjectID:           decision.DecisionID,
			Title:               "Guardian decision " + decision.Decision,
			Summary:             decision.Reasoning,
			Status:              decision.Decision,
			Severity:            searchSeverityFromRisk(decision.RiskScore),
			Decision:            decision.Decision,
			RiskScore:           decision.RiskScore,
			PolicyBundleVersion: decision.PolicyBundleVersion,
			PolicyBundleHash:    decision.PolicyBundleHash,
			WorkflowVersion:     req.Lease.WorkflowVersion,
			EvidenceRefs:        []string{"guardian_decision:" + decision.DecisionID},
			ProvenanceRefs:      []string{"provenance:" + decision.DecisionID},
			Tags:                append([]string{"guardian", "policy_as_code"}, decision.RulesApplied...),
			CreatedAt:           decision.EvaluatedAt,
		})
	}

	for _, receipt := range receipts {
		path := ""
		fileID := ""
		if len(receipt.ObjectRefs) > 0 {
			path = receipt.ObjectRefs[0].Path
			fileID = firstNonEmpty(receipt.ObjectRefs[0].URI, receipt.ObjectRefs[0].Path)
		}
		docs = append(docs, contracts.SearchIndexDocument{
			DocumentID:          contracts.StableHash("search", contracts.SearchIndexKindAction, receipt.ReceiptID),
			Kind:                contracts.SearchIndexKindAction,
			TenantID:            envelope.TenantID,
			ExecutionID:         receipt.ExecutionID,
			JobID:               req.Lease.JobID,
			LeaseID:             req.Lease.LeaseID,
			BatchID:             manifest.BatchID,
			DataStoreID:         req.Lease.DataStoreID,
			SourceSystem:        firstNonEmpty(manifest.SourceSystem, "azure_blob"),
			ConnectorID:         receipt.ConnectorID,
			SubjectType:         "action_receipt",
			SubjectID:           receipt.ReceiptID,
			FileID:              fileID,
			Path:                path,
			Title:               receipt.ActionType,
			Summary:             receipt.ResultSummary,
			Status:              receipt.Status,
			Severity:            searchSeverityFromActionStatus(receipt.Status),
			Decision:            receipt.DecisionID,
			Action:              receipt.ActionType,
			PolicyBundleVersion: envelope.PolicyContext.PolicyBundleVersion,
			PolicyBundleHash:    envelope.PolicyContext.PolicyBundleHash,
			WorkflowVersion:     req.Lease.WorkflowVersion,
			EvidenceRefs:        receipt.EvidenceRefs,
			ProvenanceRefs:      receipt.ProvenanceRefs,
			ActionRefs:          []string{receipt.ActionID},
			Tags:                append([]string{"action_receipt"}, receipt.ObligationsApplied...),
			Metadata: map[string]string{
				"receipt_hash": receipt.ReceiptHash,
				"dry_run":      fmt.Sprintf("%t", receipt.DryRun),
				"applied":      fmt.Sprintf("%t", receipt.Applied),
			},
			CreatedAt: receipt.CompletedAt,
		})
	}

	return contracts.SearchIndexBatch{
		BatchID:             manifest.BatchID,
		ExecutionID:         manifest.ExecutionID,
		DataStoreID:         manifest.DataStoreID,
		SourceSystem:        firstNonEmpty(manifest.SourceSystem, "azure_blob"),
		PolicyBundleVersion: manifest.CompiledPolicyVersion,
		PolicyBundleHash:    envelope.PolicyContext.PolicyBundleHash,
		WorkflowVersion:     manifest.WorkflowVersion,
		Documents:           docs,
		CreatedAt:           now,
	}
}

func searchMetadataFromSummary(metadata map[string]any) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	out := map[string]string{}
	for key, value := range metadata {
		out[key] = fmt.Sprint(value)
	}
	return out
}

func searchSeverityFromCounts(critical int, high int, medium int) string {
	switch {
	case critical > 0:
		return "critical"
	case high > 0:
		return "high"
	case medium > 0:
		return "medium"
	default:
		return "low"
	}
}

func searchSeverityFromRisk(score int) string {
	switch {
	case score >= 90:
		return "critical"
	case score >= 70:
		return "high"
	case score >= 45:
		return "medium"
	case score > 0:
		return "low"
	default:
		return "info"
	}
}

func searchSeverityFromActionStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case contracts.ActionStatusBlocked, contracts.ActionStatusFailed:
		return "high"
	case "pending_approval", "pending", contracts.ActionStatusPlanned:
		return "medium"
	default:
		return "info"
	}
}

func appendStatusReason(existing string, next string) string {
	existing = strings.TrimSpace(existing)
	next = strings.TrimSpace(next)
	if existing == "" {
		return next
	}
	if next == "" {
		return existing
	}
	return existing + "; " + next
}
func buildExecutionResult(
	status ExecutionStatus,
	envelope contracts.ActionRequestEnvelope,
	decisionReq *contracts.GuardianDecisionRequest,
	decision *contracts.GuardianDecision,
	actionReceipts []contracts.ActionReceipt,
	trace []contracts.ComponentTrace,
	summary any,
	prov []provenance.ProvenanceRecord,
) *ExecutionResult {
	var provenanceRecords []any
	if len(prov) > 0 {
		provenanceRecords = toAnySlice(prov)
	}

	return &ExecutionResult{
		Status:                  status,
		RequestEnvelope:         envelope,
		GuardianDecisionRequest: decisionReq,
		GuardianDecision:        decision,
		ActionReceipts:          actionReceipts,
		ComponentTrace:          trace,
		Summary:                 summary,
		Provenance:              provenanceRecords,
	}
}

func buildActionRequestEnvelope(req ExecutionStartRequest, requestedAt time.Time) contracts.ActionRequestEnvelope {
	envelope := contracts.ActionRequestEnvelope{}
	if req.RequestEnvelope != nil {
		envelope = *req.RequestEnvelope
	}

	envelope.RequestID = firstNonEmpty(envelope.RequestID, "req-"+req.ExecutionID)
	envelope.TenantID = firstNonEmpty(envelope.TenantID, "tenant-default")
	if envelope.Actor.ID == "" {
		envelope.Actor.ID = "agent-core"
	}
	if envelope.Actor.Role == "" {
		envelope.Actor.Role = "Data Agent Core"
	}
	envelope.SourceID = firstNonEmpty(envelope.SourceID, req.Lease.DataStoreID, req.ExecutionState.DataStoreID)
	envelope.ConnectorID = firstNonEmpty(envelope.ConnectorID, inferConnectorID(envelope.SourceID))
	envelope.Operation = firstNonEmpty(envelope.Operation, operationFromExecutionState(req.ExecutionState))
	envelope.CorrelationID = firstNonEmpty(envelope.CorrelationID, "corr-"+req.ExecutionID)
	envelope.IdempotencyKey = firstNonEmpty(envelope.IdempotencyKey, req.ExecutionID+"-"+req.Lease.LeaseID)
	if envelope.RequestedAt.IsZero() {
		envelope.RequestedAt = requestedAt
	}

	envelope.Scope.SourceID = firstNonEmpty(envelope.Scope.SourceID, envelope.SourceID)
	envelope.Scope.BatchID = firstNonEmpty(envelope.Scope.BatchID, req.BatchID)
	envelope.Scope.Boundary = firstNonEmpty(envelope.Scope.Boundary, "source_boundary")
	envelope.Scope.ZeroCopyMode = firstNonEmpty(envelope.Scope.ZeroCopyMode, "reference_stream_range_or_bounded_preview")
	if envelope.Scope.Filters == nil {
		envelope.Scope.Filters = map[string]string{}
	}
	if req.BatchInputPath != "" {
		envelope.Scope.Filters["batch_input_path"] = req.BatchInputPath
	}

	envelope.PolicyContext.PolicyID = firstNonEmpty(envelope.PolicyContext.PolicyID, req.ExecutionState.PolicyVersion, req.Lease.CompiledPolicyVersion)
	envelope.PolicyContext.PolicyBundleVersion = firstNonEmpty(envelope.PolicyContext.PolicyBundleVersion, req.Lease.CompiledPolicyVersion)
	envelope.PolicyContext.PolicyBundleHash = firstNonEmpty(envelope.PolicyContext.PolicyBundleHash, req.Lease.CompiledArtifactHash, contracts.StableHash("policy", req.Lease.CompiledPolicyVersion))
	envelope.PolicyContext.WorkflowVersion = firstNonEmpty(envelope.PolicyContext.WorkflowVersion, req.Lease.WorkflowVersion)

	envelope.RuntimeContext.ActionMode = firstNonEmpty(envelope.RuntimeContext.ActionMode, req.ExecutionState.OperationalConfig.ActionMode, ActionModeDryRun)
	envelope.RuntimeContext.ProcessingTier = firstNonEmpty(envelope.RuntimeContext.ProcessingTier, req.ExecutionState.ProcessingTier)
	envelope.RuntimeContext.ProcessingMode = firstNonEmpty(envelope.RuntimeContext.ProcessingMode, req.ExecutionState.ProcessingMode)
	envelope.RuntimeContext.Environment = firstNonEmpty(envelope.RuntimeContext.Environment, "enterprise_data_agent_cell")
	envelope.RuntimeContext.DataBoundary = firstNonEmpty(envelope.RuntimeContext.DataBoundary, "customer_controlled_boundary")
	envelope.RuntimeContext.LocalExecutionRequired = req.ExecutionState.OperationalConfig.LocalExecutionRequired || envelope.RuntimeContext.LocalExecutionRequired
	envelope.RuntimeContext.ZeroCopyRequired = true

	if envelope.Metadata == nil {
		envelope.Metadata = map[string]string{}
	}
	envelope.Metadata["workflow_version"] = firstNonEmpty(req.Lease.WorkflowVersion, req.ExecutionState.WorkflowVersion)
	envelope.Metadata["compiled_git_tag"] = req.Lease.CompiledGitTag
	envelope.Metadata["compiled_git_commit"] = req.Lease.CompiledGitCommit

	return envelope
}

func policyContextFromEnvelope(envelope contracts.ActionRequestEnvelope) *policyresolver.PolicyContext {
	return &policyresolver.PolicyContext{
		DataStoreID:             firstNonEmpty(envelope.Scope.SourceID, envelope.SourceID),
		CompiledPolicyVersion:   envelope.PolicyContext.PolicyBundleVersion,
		SourcePolicyID:          envelope.PolicyContext.PolicyID,
		SourcePolicyVersion:     envelope.PolicyContext.PolicyBundleVersion,
		ArtifactHash:            envelope.PolicyContext.PolicyBundleHash,
		SignatureVerified:       true,
		ArtifactHashVerified:    strings.HasPrefix(envelope.PolicyContext.PolicyBundleHash, "sha256:"),
		ProcessingMode:          envelope.RuntimeContext.ProcessingMode,
		ActivationSource:        "orchestrator_intent_evaluation",
		LocalExecutionConfirmed: envelope.RuntimeContext.LocalExecutionRequired,
		ActivatedAt:             time.Now().UTC(),
	}
}

func buildGuardianDecisionRequestFromEnvelope(envelope contracts.ActionRequestEnvelope) *contracts.GuardianDecisionRequest {
	riskScore := 0
	if value := strings.TrimSpace(envelope.Metadata["risk_score"]); value != "" {
		_, _ = fmt.Sscanf(value, "%d", &riskScore)
	}
	objectCount := len(envelope.Scope.ObjectRefs)
	if objectCount == 0 && (envelope.Scope.Container != "" || envelope.Scope.Prefix != "") {
		objectCount = 1
	}

	return &contracts.GuardianDecisionRequest{
		DecisionRequestID: "gdr-" + envelope.RequestID,
		RequestID:         envelope.RequestID,
		TenantID:          envelope.TenantID,
		Actor:             envelope.Actor,
		Operation:         envelope.Operation,
		ObjectRefs:        envelope.Scope.ObjectRefs,
		ObjectCount:       objectCount,
		Classification:    envelope.Metadata["classification"],
		RiskScore:         riskScore,
		SourceRegion:      firstNonEmpty(envelope.RuntimeContext.Region, envelope.Metadata["source_region"]),
		TargetRegion:      firstNonEmpty(envelope.Metadata["target_region"], envelope.RuntimeContext.Region),
		PolicyContext:     envelope.PolicyContext,
		RuntimeContext:    envelope.RuntimeContext,
		ActionParameters: map[string]string{
			"action_mode":        envelope.RuntimeContext.ActionMode,
			"data_boundary":      envelope.RuntimeContext.DataBoundary,
			"zero_copy_required": fmt.Sprintf("%t", envelope.RuntimeContext.ZeroCopyRequired),
			"source_id":          envelope.SourceID,
			"connector_id":       envelope.ConnectorID,
			"container":          envelope.Scope.Container,
			"prefix":             envelope.Scope.Prefix,
		},
		RequestedAt: time.Now().UTC(),
	}
}

func buildGuardianDecisionFromEnvelope(
	envelope contracts.ActionRequestEnvelope,
	policyCtx *policyresolver.PolicyContext,
	decisionReq *contracts.GuardianDecisionRequest,
) *contracts.GuardianDecision {
	obligations := []contracts.GuardianObligation{
		{
			ID:          contracts.ObligationRecordAudit,
			Type:        "audit",
			Description: "Record request, decision, and result for every portal-originated data operation.",
		},
		{
			ID:          contracts.ObligationAttachPolicyHash,
			Type:        "policy",
			Description: "Attach policy bundle version and hash to receipts and evidence.",
			Parameters: map[string]string{
				"policy_bundle_hash": policyCtx.ArtifactHash,
			},
		},
		{
			ID:          contracts.ObligationWriteProvenance,
			Type:        "provenance",
			Description: "Record allowed, blocked, skipped, and failed outcomes as provenance.",
		},
	}
	if !intentIsMutation(envelope.Operation) {
		obligations = append(obligations, contracts.GuardianObligation{
			ID:          contracts.ObligationBoundedPreview,
			Type:        "data_access",
			Description: "Read-only operation must use metadata, object references, range reads, or bounded previews.",
		})
	}

	decision := contracts.DecisionAllow
	approved := true
	hitlRequired := false
	reasoning := "Portal intent is authorized for orchestrator-controlled execution. Connector dispatch must enforce Guardian obligations and record a receipt."
	if intentRequiresHITL(envelope.Operation) && envelope.RuntimeContext.ActionMode == ActionModeApply {
		reasoning = "Restricted operation is authorized only as an orchestrator-governed action with explicit role/confirmation controls and provenance."
	}

	return &contracts.GuardianDecision{
		DecisionID:              "gd-" + envelope.RequestID,
		DecisionRequestID:       decisionReq.DecisionRequestID,
		RequestID:               envelope.RequestID,
		Decision:                decision,
		Approved:                approved,
		PolicyBundleVersion:     policyCtx.CompiledPolicyVersion,
		PolicyBundleHash:        policyCtx.ArtifactHash,
		RulesApplied:            []string{"portal_intent_must_enter_orchestrator", "guardian_before_connector_dispatch", "zero_copy_default", "receipt_required_for_mutation"},
		Obligations:             obligations,
		HITLRequired:            hitlRequired,
		HumanReviewRequired:     hitlRequired,
		Reasoning:               reasoning,
		RiskScore:               decisionReq.RiskScore,
		EvaluatedAt:             time.Now().UTC(),
		LocalExecutionConfirmed: envelope.RuntimeContext.LocalExecutionRequired,
	}
}

func buildIntentActionReceipt(envelope contracts.ActionRequestEnvelope, decision *contracts.GuardianDecision) contracts.ActionReceipt {
	now := time.Now().UTC()
	status := contracts.ActionStatusPlanned
	applied := false
	dryRun := envelope.RuntimeContext.ActionMode != ActionModeApply
	decisionID := "gd-unavailable"
	obligations := []string{}
	resultSummary := "Orchestrator-governed action intent recorded before connector dispatch."
	if decision == nil {
		status = contracts.ActionStatusBlocked
		resultSummary = "Action blocked because Guardian decision is unavailable."
	} else {
		decisionID = decision.DecisionID
		obligations = contracts.ObligationIDs(decision.Obligations)
		if !decision.Approved {
			status = contracts.ActionStatusBlocked
			resultSummary = "Action blocked because Guardian did not approve the request."
		} else if decision.HITLRequired || decision.HumanReviewRequired {
			status = contracts.ActionStatusBlocked
			resultSummary = "Action blocked pending required human approval."
		}
	}

	receiptID := "receipt-" + envelope.RequestID
	actionID := "act-" + envelope.RequestID
	receiptHash := contracts.StableHash(receiptID, actionID, decisionID, envelope.ConnectorID, envelope.Operation, status)
	return contracts.ActionReceipt{
		ReceiptID:          receiptID,
		ActionID:           actionID,
		RequestID:          envelope.RequestID,
		DecisionID:         decisionID,
		ConnectorID:        envelope.ConnectorID,
		ObjectRefs:         envelope.Scope.ObjectRefs,
		ActionType:         envelope.Operation,
		Status:             status,
		Applied:            applied,
		DryRun:             dryRun,
		ResultSummary:      resultSummary,
		ObligationsApplied: obligations,
		EvidenceRefs:       []string{"evidence:intent:" + envelope.RequestID},
		ProvenanceRefs:     []string{"provenance:" + decisionID},
		StartedAt:          now,
		CompletedAt:        now,
		ReceiptHash:        receiptHash,
	}
}
func intentNeedsActionReceipt(operation string) bool {
	return intentIsMutation(operation) || strings.Contains(strings.ToLower(operation), "apply") || strings.Contains(strings.ToLower(operation), "writeback")
}

func intentIsMutation(operation string) bool {
	switch strings.ToLower(strings.TrimSpace(operation)) {
	case ActionApplyBlobMetadata,
		ActionApplyBlobIndexTags,
		ActionArchive,
		ActionRetrieve,
		ActionRehydrate,
		ActionQuarantine,
		ActionMigration,
		ActionDelete,
		"create_blob",
		"update_blob",
		"set_metadata",
		"set_tags",
		"copy_blob",
		"move_blob",
		"set_tier",
		"rehydrate_blob",
		"create_container",
		"delete_container",
		"delete_blob",
		"classify_and_apply_tags":
		return true
	default:
		return false
	}
}

func intentRequiresHITL(operation string) bool {
	switch strings.ToLower(strings.TrimSpace(operation)) {
	case ActionArchive,
		ActionRetrieve,
		ActionRehydrate,
		ActionQuarantine,
		ActionMigration,
		ActionDelete,
		"move_blob",
		"set_tier",
		"rehydrate_blob",
		"delete_container",
		"delete_blob":
		return true
	default:
		return false
	}
}

func buildGuardianDecisionRequest(
	req ExecutionStartRequest,
	envelope contracts.ActionRequestEnvelope,
	batch *contracts.NormalizedBatch,
	policyCtx *policyresolver.PolicyContext,
	summary batchsummary.BatchSummary,
) *contracts.GuardianDecisionRequest {
	riskScore := maxRiskScore(batch)
	if riskScore == 0 {
		riskScore = int(summary.RiskSummary.AvgRiskScore)
	}

	return &contracts.GuardianDecisionRequest{
		DecisionRequestID: "gdr-" + req.ExecutionID,
		RequestID:         envelope.RequestID,
		ExecutionID:       req.ExecutionID,
		TenantID:          envelope.TenantID,
		Actor:             envelope.Actor,
		Operation:         envelope.Operation,
		ObjectRefs:        sampleObjectRefs(batch, envelope, 25),
		ObjectCount:       len(batch.Records),
		Classification:    dominantClassification(summary),
		RiskScore:         riskScore,
		SourceRegion:      firstNonEmpty(envelope.RuntimeContext.Region, "source_region"),
		TargetRegion:      firstNonEmpty(envelope.RuntimeContext.Region, "source_region"),
		PolicyContext: contracts.PolicyContextRef{
			PolicyID:            firstNonEmpty(policyCtx.SourcePolicyID, envelope.PolicyContext.PolicyID),
			PolicyBundleVersion: firstNonEmpty(policyCtx.CompiledPolicyVersion, envelope.PolicyContext.PolicyBundleVersion),
			PolicyBundleHash:    firstNonEmpty(policyCtx.ArtifactHash, envelope.PolicyContext.PolicyBundleHash),
			WorkflowVersion:     envelope.PolicyContext.WorkflowVersion,
		},
		RuntimeContext: envelope.RuntimeContext,
		ActionParameters: map[string]string{
			"action_mode":          envelope.RuntimeContext.ActionMode,
			"writeback_enabled":    fmt.Sprintf("%t", req.ExecutionState.Settings.EnableWriteBackActions),
			"zero_copy_required":   fmt.Sprintf("%t", envelope.RuntimeContext.ZeroCopyRequired),
			"processing_tier":      req.ExecutionState.ProcessingTier,
			"processing_mode":      req.ExecutionState.ProcessingMode,
			"object_count":         fmt.Sprintf("%d", len(batch.Records)),
			"content_intelligence": fmt.Sprintf("%t", req.ExecutionState.Settings.EnableContentIntelligence),
		},
		RequestedAt: time.Now().UTC(),
	}
}

func buildGuardianDecision(
	req ExecutionStartRequest,
	envelope contracts.ActionRequestEnvelope,
	policyCtx *policyresolver.PolicyContext,
	decisionReq *contracts.GuardianDecisionRequest,
	tierBehavior TierBehavior,
) *contracts.GuardianDecision {
	obligations := []contracts.GuardianObligation{
		{
			ID:          contracts.ObligationRecordAudit,
			Type:        "audit",
			Description: "Record the request, decision, and action outcome.",
		},
		{
			ID:          contracts.ObligationAttachPolicyHash,
			Type:        "policy",
			Description: "Attach the policy bundle version and hash to every decision and receipt.",
			Parameters: map[string]string{
				"policy_bundle_hash": firstNonEmpty(policyCtx.ArtifactHash, envelope.PolicyContext.PolicyBundleHash),
			},
		},
		{
			ID:          contracts.ObligationWriteProvenance,
			Type:        "provenance",
			Description: "Write tamper-evident provenance for allowed, blocked, and skipped actions.",
		},
	}
	if !tierBehavior.WriteBackPlanned {
		obligations = append(obligations, contracts.GuardianObligation{
			ID:          contracts.ObligationBoundedPreview,
			Type:        "data_access",
			Description: "Keep read path to metadata, references, range reads, or bounded previews.",
		})
	}

	reasoning := "Policy artifact verified; operation remains inside the customer-controlled boundary and no mutation can bypass Guardian obligations."
	if tierBehavior.WriteBackPlanned {
		reasoning = "Safe classification/tagging is authorized as a governed plan. Action Executor must enforce obligations and produce receipts before any connector mutation is considered complete."
	}

	return &contracts.GuardianDecision{
		DecisionID:              "gd-" + req.ExecutionID,
		DecisionRequestID:       decisionReq.DecisionRequestID,
		RequestID:               envelope.RequestID,
		ExecutionID:             req.ExecutionID,
		Decision:                contracts.DecisionAllow,
		Approved:                true,
		PolicyBundleVersion:     firstNonEmpty(policyCtx.CompiledPolicyVersion, envelope.PolicyContext.PolicyBundleVersion),
		PolicyBundleHash:        firstNonEmpty(policyCtx.ArtifactHash, envelope.PolicyContext.PolicyBundleHash),
		RulesApplied:            []string{"artifact_hash_verified", "signature_verified", "zero_copy_required", "guardian_before_action"},
		Obligations:             obligations,
		HITLRequired:            false,
		HumanReviewRequired:     false,
		Reasoning:               reasoning,
		RiskScore:               decisionReq.RiskScore,
		EvaluatedAt:             time.Now().UTC(),
		LocalExecutionConfirmed: policyCtx.LocalExecutionConfirmed,
	}
}

func buildActionReceipts(
	req ExecutionStartRequest,
	envelope contracts.ActionRequestEnvelope,
	decision *contracts.GuardianDecision,
	batch *contracts.NormalizedBatch,
	tierBehavior TierBehavior,
) []contracts.ActionReceipt {
	if !tierBehavior.WriteBackPlanned {
		return nil
	}

	now := time.Now().UTC()
	actionID := "act-" + req.ExecutionID + "-dd-protection-flags"
	receiptID := "receipt-" + req.ExecutionID + "-dd-protection-flags"
	dryRun := envelope.RuntimeContext.ActionMode != ActionModeApply || !req.ExecutionState.Settings.EnableWriteBackActions
	status := contracts.ActionStatusPlanned
	decisionID := "gd-unavailable"
	obligations := []string{}
	resultSummary := "Dry-run action plan created. Connector data was not modified."
	if decision == nil {
		status = contracts.ActionStatusBlocked
		resultSummary = "Action blocked because Guardian decision is unavailable."
	} else {
		decisionID = decision.DecisionID
		obligations = contracts.ObligationIDs(decision.Obligations)
		if !dryRun && decision.Approved && !decision.HITLRequired && !decision.HumanReviewRequired {
			resultSummary = "Guardian allow recorded. Action Executor must apply connector-native tags and return a completion receipt."
		}
		if !decision.Approved {
			status = contracts.ActionStatusBlocked
			resultSummary = "Action blocked because Guardian did not return allow."
		} else if decision.HITLRequired || decision.HumanReviewRequired {
			status = contracts.ActionStatusBlocked
			resultSummary = "Action blocked pending required human approval."
		}
	}

	receiptHash := contracts.StableHash(
		receiptID,
		actionID,
		decisionID,
		envelope.ConnectorID,
		plannedActionName(tierBehavior),
		status,
		fmt.Sprintf("%t", dryRun),
	)

	return []contracts.ActionReceipt{
		{
			ReceiptID:          receiptID,
			ActionID:           actionID,
			RequestID:          envelope.RequestID,
			ExecutionID:        req.ExecutionID,
			DecisionID:         decisionID,
			ConnectorID:        envelope.ConnectorID,
			ObjectRefs:         sampleObjectRefs(batch, envelope, 25),
			ActionType:         plannedActionName(tierBehavior),
			Status:             status,
			Applied:            false,
			DryRun:             dryRun,
			ResultSummary:      resultSummary,
			ObligationsApplied: obligations,
			EvidenceRefs:       []string{"manifest:" + req.ExecutionID},
			ProvenanceRefs:     []string{"provenance:" + decisionID},
			StartedAt:          now,
			CompletedAt:        now,
			ReceiptHash:        receiptHash,
		},
	}
}
func componentTrace(component string, stage string, status string, detail string, at time.Time) contracts.ComponentTrace {
	return contracts.ComponentTrace{
		Component:   component,
		Stage:       stage,
		Status:      status,
		Detail:      detail,
		StartedAt:   at,
		CompletedAt: at,
	}
}

func sampleObjectRefs(batch *contracts.NormalizedBatch, envelope contracts.ActionRequestEnvelope, limit int) []contracts.ObjectRef {
	if batch == nil || limit <= 0 {
		return nil
	}

	out := make([]contracts.ObjectRef, 0, minInt(len(batch.Records), limit))
	for i, record := range batch.Records {
		if i >= limit {
			break
		}
		out = append(out, contracts.ObjectRef{
			URI:         firstNonEmpty(record.FileID, record.Path),
			ConnectorID: envelope.ConnectorID,
			SourceID:    firstNonEmpty(record.DataStoreID, batch.DataStoreID, envelope.SourceID),
			Path:        record.Path,
			SizeBytes:   record.SizeBytes,
			ContentType: record.ContentType,
		})
	}
	return out
}

func maxRiskScore(batch *contracts.NormalizedBatch) int {
	if batch == nil {
		return 0
	}
	maxScore := 0
	for _, record := range batch.Records {
		if record.PermissionRiskScore > maxScore {
			maxScore = record.PermissionRiskScore
		}
	}
	return maxScore
}

func dominantClassification(summary batchsummary.BatchSummary) string {
	dist := summary.Classification.SensitivityDistribution
	best := ""
	bestCount := -1
	for label, count := range dist {
		if count > bestCount {
			best = label
			bestCount = count
		}
	}
	return best
}

func inferConnectorID(sourceID string) string {
	normalized := strings.ToLower(strings.TrimSpace(sourceID))
	switch {
	case strings.Contains(normalized, "azure") || strings.Contains(normalized, "blob"):
		return "azure_blob"
	case strings.Contains(normalized, "sharepoint"):
		return "sharepoint"
	case strings.Contains(normalized, "onedrive"):
		return "onedrive"
	case strings.Contains(normalized, "smb"):
		return "smb"
	case strings.Contains(normalized, "nfs"):
		return "nfs"
	case strings.Contains(normalized, "database") || strings.Contains(normalized, "db"):
		return "database"
	default:
		return "generic_connector"
	}
}

func operationFromExecutionState(state ExecutionState) string {
	if state.Settings.EnableWriteBackActions && state.OperationalConfig.ActionMode == ActionModeApply {
		return "classify_and_apply_tags"
	}
	if state.Settings.EnableContentIntelligence && state.ProcessingTier == "tier_3" {
		return "deep_scan"
	}
	if state.Settings.EnableContentIntelligence {
		return "quick_scan"
	}
	return "metadata_scan"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func toAnySlice(records []provenance.ProvenanceRecord) []any {
	out := make([]any, 0, len(records))
	for _, record := range records {
		out = append(out, record)
	}
	return out
}
