package constants

import "fmt"

const (
	// AppName is the name of the application.
	AppName = "tssc"

	// Namespace is the default namespace for the application.
	Namespace = "tssc"

	// OrgName is the name of the organization.
	OrgName = "redhat-appstudio"

	// Domain organization domain.
	Domain = "github.com"

	// ConfigFilename is the name of the configuration file.
	ConfigFilename = "config.yaml"

	// ValuesFilename is the name of the values template file.
	ValuesFilename = "values.yaml.tpl"

	// InstructionsFilename is the name of the instructions file.
	InstructionsFilename = "instructions.md"
)

var (
	// RepoURI is the reverse repository URI for the application.
	RepoURI = fmt.Sprintf("%s.%s.%s", AppName, OrgName, Domain)
)
