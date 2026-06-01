package execution

import (
	"fmt"
	"time"

	"everest.local/data-agent-policy-resolver/internal/contracts"
)

func buildManifest(
	req ExecutionStartRequest,
	status ExecutionStatus,
	batch *contracts.NormalizedBatch,
) (contracts.ManifestSummary, []contracts.ManifestDetail) {
	summary := contracts.ManifestSummary{
		ExecutionID:           status.ExecutionID,
		JobID:                 status.JobID,
		LeaseID:               status.LeaseID,
		BatchID:               batch.BatchID,
		DataStoreID:           status.DataStoreID,
		SourceSystem:          sourceSystem(batch),
		Status:                status.Status,
		TotalRecords:          len(batch.Records),
		ProcessedRecords:      status.RecordsProcessed,
		CompiledPolicyVersion: status.CompiledPolicyVersion,
		WorkflowVersion:       status.WorkflowVersion,
		ProcessingTier:        status.ProcessingTier,
		ProcessingMode:        req.ExecutionState.ProcessingMode,
		StartedAt:             status.StartedAt.Format(time.RFC3339),
		CompletedAt:           status.CompletedAt.Format(time.RFC3339),
	}

	details := make([]contracts.ManifestDetail, 0, len(batch.Records))

	totalRisk := 0

	for _, record := range batch.Records {
		riskScore := record.PermissionRiskScore
		riskLevel := riskLevelFromScore(riskScore)
		classification := classificationFromRiskLevel(riskLevel)

		totalRisk += riskScore
		summary.TotalSizeBytes += record.SizeBytes

		switch riskLevel {
		case "critical":
			summary.CriticalCount++
		case "high":
			summary.HighCount++
		case "medium":
			summary.MediumCount++
		default:
			summary.LowCount++
		}

		actionRequested := "none"
		actionStatus := "not_applicable"
		guardianDecision := "not_evaluated"
		provenanceID := ""

		if riskLevel == "critical" || riskLevel == "high" {
			actionRequested = "dry_run_apply_metadata_tag"
			actionStatus = "planned_dry_run"
			guardianDecision = "pending_real_guardian"
			provenanceID = fmt.Sprintf("prov-%s", record.FileID)
			summary.ActionsPlanned++
		}

		details = append(details, contracts.ManifestDetail{
			ExecutionID:       status.ExecutionID,
			BatchID:           batch.BatchID,
			DataStoreID:       status.DataStoreID,
			FileID:            record.FileID,
			Path:              record.Path,
			Name:              record.Name,
			SizeBytes:         record.SizeBytes,
			ContentType:       record.ContentType,
			Extension:         record.Extension,
			SourceSystem:      record.SourceSystem,
			RiskScore:         riskScore,
			RiskLevel:         riskLevel,
			Classification:    classification,
			Entities:          []string{},
			GuardianDecision:  guardianDecision,
			ActionRequested:   actionRequested,
			ActionStatus:      actionStatus,
			Error:             "",
			ProvenanceID:      provenanceID,
			CompiledPolicyVer: status.CompiledPolicyVersion,
			WorkflowVersion:   status.WorkflowVersion,
		})
	}

	if len(batch.Records) > 0 {
		summary.AverageRiskScore = float64(totalRisk) / float64(len(batch.Records))
	}

	return summary, details
}

func sourceSystem(batch *contracts.NormalizedBatch) string {
	if batch == nil || len(batch.Records) == 0 {
		return ""
	}
	return batch.Records[0].SourceSystem
}

func riskLevelFromScore(score int) string {
	switch {
	case score >= 90:
		return "critical"
	case score >= 70:
		return "high"
	case score >= 40:
		return "medium"
	default:
		return "low"
	}
}

func classificationFromRiskLevel(level string) string {
	switch level {
	case "critical", "high":
		return "confidential"
	case "medium":
		return "internal"
	default:
		return "public"
	}
}
