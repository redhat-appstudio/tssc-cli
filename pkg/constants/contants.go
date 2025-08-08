package constants

import "fmt"

const (
	// AppName is the name of the application.
	AppName = "tssc"

	// OrgName is the name of the organization.
	OrgName = "redhat-appstudio"

	// Domain organization domain.
	Domain = "github.com"
)

var (
	// RepoURI is the reverse repository URI for the application.
	RepoURI = fmt.Sprintf("%s.%s.%s", AppName, OrgName, Domain)
)
