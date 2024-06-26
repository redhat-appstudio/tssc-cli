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
	// RedHatDeveloperHub Red Hat Developer Hub (RHDH).
	RedHatDeveloperHub = "redHatDeveloperHub"
	// RedHatAdvancedClusterSecurity Red Hat Advanced Cluster Security (RHACS).
	RedHatAdvancedClusterSecurity = "redHatAdvancedClusterSecurity"
	// RedHatQuay Red Hat Quay (RHDH).
	RedHatQuay = "redHatQuay"
	// OpenShiftPipelines OpenShift Pipelines.
	OpenShiftPipelines = "openShiftPipelines"
)

// FeatureSpec contains the configuration for a specific feature.
type FeatureSpec struct {
	// Enabled feature toggle.
	Enabled bool `yaml:"enabled"`
	// Namespace target namespace for the feature, which may involve different
	// Helm charts targeting the specific feature namespace, while the chart
	// target is deployed in a different namespace.
	Namespace *string `yaml:"namespace,omitempty"`
	// Properties contains the feature specific configuration.
	Properties map[string]interface{} `yaml:"properties"`
}

// GetNamespace returns the feature namespace, or an empty string if not set.
func (f *FeatureSpec) GetNamespace() string {
	if f.Namespace == nil {
		return ""
	}
	return *f.Namespace
}

// Validate validates the feature configuration, checking for missing fields.
func (f *FeatureSpec) Validate() error {
	if f.Enabled && f.GetNamespace() == "" {
		return fmt.Errorf("%w: missing namespace", ErrInvalidConfig)
	}
	return nil
}
