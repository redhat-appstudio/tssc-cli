package annotations

// RepoURI is the reverse domain notation URI used as prefix for all
// annotations and labels managed by this application.
const RepoURI = "helmet.redhat-appstudio.github.com"

// Annotation keys for Helm chart metadata.
const (
	ProductName          = RepoURI + "/product-name"
	DependsOn            = RepoURI + "/depends-on"
	// DependsOnGlobalCharts lists charts under the installer charts/ directory (global charts).
	DependsOnGlobalCharts = RepoURI + "/depends-on-global-charts"
	// DependsOnBundleCharts lists sibling charts in the same bundles/<id>/charts/ directory.
	DependsOnBundleCharts = RepoURI + "/depends-on-bundle-charts"
	// DependsOnBundles lists bundle directory names (bundles/<id>) whose charts must be ordered before this chart.
	DependsOnBundles = RepoURI + "/depends-on-bundles"
	Weight               = RepoURI + "/weight"
	UseProductNamespace  = RepoURI + "/use-product-namespace"
	// InstallReleaseInInstallerNamespace, when "true", forces the Helm release
	// namespace to the installer namespace even when the chart is tied to a product
	// that uses a different product namespace. Use when chart templates place
	// resources (e.g. integration Secrets) in the installer namespace so Helm
	// ownership metadata stays consistent.
	InstallReleaseInInstallerNamespace = RepoURI + "/install-release-in-installer-namespace"
	IntegrationsProvided = RepoURI + "/integrations-provided"
	// InstallerIntegrationID documents a chart id suffix (local://<id> in helmet.yaml;
	// must match Chart name tssc-<id> when set). Optional.
	InstallerIntegrationID = RepoURI + "/installer-integration-id"
	// IntegrationDisplayName is an optional human-readable title (defaults to Chart description).
	IntegrationDisplayName = RepoURI + "/integration-display-name"
	IntegrationsRequired = RepoURI + "/integrations-required"
	// BundleTypesSupported lists product vs legacy integration tokens (comma-separated,
	// or the keyword "both"). Only product placement is used for Helm installs.
	BundleTypesSupported = RepoURI + "/bundle-types-supported"
	// BundleType is legacy; use BundleTypesSupported. Values: integration, product, dual.
	BundleType = RepoURI + "/bundle-type"
	PostDeploy = RepoURI + "/post-deploy"
	Config               = RepoURI + "/config"
)
