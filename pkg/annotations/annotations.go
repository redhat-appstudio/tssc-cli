package annotations

// RepoURI is the reverse domain notation URI used as prefix for all
// annotations and labels managed by this application.
const RepoURI = "helmet.redhat-appstudio.github.com"

// Annotation keys for Helm chart metadata
const (
	ProductName          = RepoURI + "/product-name"
	DependsOn            = RepoURI + "/depends-on"
	Weight               = RepoURI + "/weight"
	UseProductNamespace  = RepoURI + "/use-product-namespace"
	IntegrationsProvided = RepoURI + "/integrations-provided"
	IntegrationsRequired = RepoURI + "/integrations-required"
	PostDeploy           = RepoURI + "/post-deploy"
	Config               = RepoURI + "/config"
)
