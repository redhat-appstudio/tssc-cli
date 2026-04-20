package annotations

// RepoURI is the reverse domain notation URI used as prefix for all
// annotations and labels managed by this application.
const RepoURI = "helmet.redhat-appstudio.github.com"

// Annotation keys for Helm chart metadata.
const (
	ProductName          = RepoURI + "/product-name"
	DependsOn            = RepoURI + "/depends-on"
	Weight               = RepoURI + "/weight"
	UseProductNamespace  = RepoURI + "/use-product-namespace"
	// InstallReleaseInInstallerNamespace, when "true", forces the Helm release
	// namespace to the installer namespace even when the chart is tied to a product
	// that uses a different product namespace. Use when chart templates place
	// resources (e.g. integration Secrets) in the installer namespace so Helm
	// ownership metadata stays consistent across product vs integration-bundle deploys.
	InstallReleaseInInstallerNamespace = RepoURI + "/install-release-in-installer-namespace"
	IntegrationsProvided = RepoURI + "/integrations-provided"
	// InstallerIntegrationID documents the installer integration id for this chart:
	// the suffix of helmet.yaml local:// references and the top-level key in
	// values.yaml used for distributed merge (must match Chart name tssc-<id>).
	// Optional; when set it must equal that suffix.
	InstallerIntegrationID = RepoURI + "/installer-integration-id"
	// IntegrationDisplayName is an optional human-readable title for this chart's
	// integration when merging installer.integrations (defaults to Chart description).
	IntegrationDisplayName = RepoURI + "/integration-display-name"
	IntegrationsRequired = RepoURI + "/integrations-required"
	// BundleTypesSupported lists which helm.yaml placements are valid: integration,
	// product, or both (comma-separated, or the keyword "both"). Prefer this over
	// BundleType.
	BundleTypesSupported = RepoURI + "/bundle-types-supported"
	// BundleType is legacy; use BundleTypesSupported. Values: integration, product, dual.
	BundleType = RepoURI + "/bundle-type"
	PostDeploy = RepoURI + "/post-deploy"
	Config               = RepoURI + "/config"
)
