package execution

import "everest.local/data-agent-policy-resolver/internal/contracts"

func LoadNormalizedBatchFile(path string) (*contracts.NormalizedBatch, error) {
	return loadNormalizedBatch(path)
}
