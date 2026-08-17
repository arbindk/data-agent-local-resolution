package execution

const (
	ActionModeDryRun = "dry_run"
	ActionModeApply  = "apply"

	ActionApplyBlobMetadata  = "apply_blob_metadata"
	ActionApplyBlobIndexTags = "apply_blob_index_tags"

	ActionArchive    = "archive"
	ActionRetrieve   = "retrieve"
	ActionRehydrate  = "rehydrate"
	ActionQuarantine = "quarantine"
	ActionMigration  = "migration"
	ActionDelete     = "delete"

	ActionDryRunDelete     = "dry_run_delete"
	ActionDryRunArchive    = "dry_run_archive"
	ActionDryRunRetrieve   = "dry_run_retrieve"
	ActionDryRunQuarantine = "dry_run_quarantine"
	ActionDryRunMigration  = "dry_run_migration"

	GuardianDecisionAllow   = "allow"
	GuardianDecisionDeny    = "deny"
	GuardianDecisionReview  = "review_required"
	GuardianDecisionPending = "pending"
)

func IsRealAzureBlobWritebackAction(action string) bool {
	switch action {
	case ActionApplyBlobMetadata, ActionApplyBlobIndexTags:
		return true
	default:
		return false
	}
}

func IsDestructiveOrRestrictedAction(action string) bool {
	switch action {
	case ActionDelete, ActionArchive, ActionRetrieve, ActionRehydrate, ActionQuarantine, ActionMigration:
		return true
	default:
		return false
	}
}

func IsAzureBlobPortalAction(action string) bool {
	switch action {
	case ActionApplyBlobMetadata,
		ActionApplyBlobIndexTags,
		ActionArchive,
		ActionRetrieve,
		ActionRehydrate,
		ActionQuarantine,
		ActionMigration,
		ActionDelete:
		return true
	default:
		return false
	}
}

func CanApplyWriteback(actionMode string, writebackEnabled bool, guardianDecision string, action string) bool {
	if actionMode != ActionModeApply {
		return false
	}

	if !writebackEnabled {
		return false
	}

	if guardianDecision != GuardianDecisionAllow {
		return false
	}

	if !IsRealAzureBlobWritebackAction(action) {
		return false
	}

	if IsDestructiveOrRestrictedAction(action) {
		return false
	}

	return true
}

func CanApplyAzureBlobPortalAction(actionMode string, writebackEnabled bool, guardianDecision string, action string) bool {
	if actionMode != ActionModeApply {
		return false
	}

	if !writebackEnabled {
		return false
	}

	if guardianDecision != GuardianDecisionAllow {
		return false
	}

	return IsAzureBlobPortalAction(action)
}
