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
)

var (
	// RepoURI is the reverse repository URI for the application.
	RepoURI = fmt.Sprintf("%s.%s.%s", AppName, OrgName, Domain)

	// Version is the application version, set at build time via ldflags.
	Version = "v0.0.0-SNAPSHOT"

	// CommitID is the commit ID of the application, set at build time via git
	// commit hash.
	CommitID = ""
)
