package api

import (
	"context"

	"everest.local/data-agent-policy-resolver/internal/connectors/azureblob"
	"everest.local/data-agent-policy-resolver/internal/contracts"
)

type azureBlobConnectorPort interface {
	NewScanner(azureblob.Config) (azureBlobScannerPort, error)
}

type azureBlobScannerPort interface {
	TestConnection(context.Context) error
	Scan(context.Context) (*contracts.NormalizedBatch, *azureblob.ScanResult, error)
	ListContainers(context.Context) ([]string, error)
	ExecuteAction(context.Context, azureblob.ActionRequest) (*azureblob.ActionResult, error)
	ApplyMetadata(context.Context, azureblob.WritebackRequest) (*azureblob.WritebackResult, error)
	ApplyIndexTags(context.Context, azureblob.WritebackRequest) (*azureblob.WritebackResult, error)
}

type defaultAzureBlobConnector struct{}

func (defaultAzureBlobConnector) NewScanner(cfg azureblob.Config) (azureBlobScannerPort, error) {
	return azureblob.NewScanner(cfg)
}

func (s *Server) newAzureBlobScanner(cfg azureblob.Config) (azureBlobScannerPort, error) {
	port := s.azureBlobConnector
	if port == nil {
		port = defaultAzureBlobConnector{}
	}
	return port.NewScanner(cfg)
}
