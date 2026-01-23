package api

// AppContext holds immutable application metadata.
// This is passed throughout the component tree as the single source of truth
// for application identity, versioning, and organizational information.
//
// AppContext is separated from App to distinguish between configuration
// (what the app is) and runtime (how the app runs).
type AppContext struct {
	Name      string // application name
	Version   string // application version
	CommitID  string // git commit ID
	Namespace string // default installation namespace
	Short     string // short description for CLI
	Long      string // long description for CLI
}

// ContextOption is a functional option for configuring AppContext.
type ContextOption func(*AppContext)

// WithNamespace sets the default installation namespace.
func WithNamespace(namespace string) ContextOption {
	return func(a *AppContext) {
		a.Namespace = namespace
	}
}

// WithVersion sets the application version.
func WithVersion(version string) ContextOption {
	return func(a *AppContext) {
		a.Version = version
	}
}

// WithCommitID sets the git commit ID.
func WithCommitID(commitID string) ContextOption {
	return func(a *AppContext) {
		a.CommitID = commitID
	}
}

// WithShortDescription sets the short CLI description.
func WithShortDescription(short string) ContextOption {
	return func(a *AppContext) {
		a.Short = short
	}
}

// WithLongDescription sets the long CLI description.
func WithLongDescription(long string) ContextOption {
	return func(a *AppContext) {
		a.Long = long
	}
}

// NewAppContext creates a new application context with sensible defaults.
// The only required parameter is the application name; all other fields
// can be configured via functional options.
func NewAppContext(name string, opts ...ContextOption) *AppContext {
	appCtx := &AppContext{
		Name:      name,
		Namespace: name,
		Version:   "v0.0.0-SNAPSHOT",
		CommitID:  "unknown",
		Short:     "",
		Long:      "",
	}
	for _, opt := range opts {
		opt(appCtx)
	}
	return appCtx
}
