package config

import (
	"fmt"
)

const (
	// CRC Code Ready Containers (CRC).
	CRC = "crc"
	// Keycloak Keycloak IAM/SSO.
	Keycloak = "keycloak"
	// TrustedProfileAnalyzer Trusted Profile Analyzer (TPA).
	TrustedProfileAnalyzer = "trustedProfileAnalyzer"
	// TrustedArtifactSigner Trusted Artifact Signer (TAS).
	TrustedArtifactSigner = "trustedArtifactSigner"
	// DeveloperHub Red Hat Developer Hub (RHDH).
	DeveloperHub = "developerHub"
	// AdvancedClusterSecurity Red Hat Advanced Cluster Security (RHACS).
	AdvancedClusterSecurity = "advancedClusterSecurity"
	// Quay Red Hat Quay (RHDH).
	Quay = "quay"
	// OpenShiftPipelines OpenShift Pipelines.
	OpenShiftPipelines = "openShiftPipelines"
)

// ProductSpec contains the configuration for a specific product.
type ProductSpec struct {
	// Enabled product toggle.
	Enabled bool `yaml:"enabled"`
	// Namespace target namespace for the product, which may involve different
	// Helm charts targeting the specific product namespace, while the chart
	// target is deployed in a different namespace.
	Namespace *string `yaml:"namespace,omitempty"`
	// Properties contains the product specific configuration.
	Properties map[string]interface{} `yaml:"properties"`
}

// GetNamespace returns the product namespace, or an empty string if not set.
func (f *ProductSpec) GetNamespace() string {
	if f.Namespace == nil {
		return ""
	}
	return *f.Namespace
}

// Validate validates the product configuration, checking for missing fields.
func (f *ProductSpec) Validate() error {
	if f.Enabled && f.GetNamespace() == "" {
		return fmt.Errorf("%w: missing namespace", ErrInvalidConfig)
	}
	return nil
}
