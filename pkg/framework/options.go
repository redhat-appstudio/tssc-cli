package framework

// Option represents a functional option for the App.
type Option func(*App)

// WithVersion sets the application version.
func WithVersion(version string) Option {
	return func(a *App) {
		a.Version = version
	}
}

// WithShortDescription sets the application short description.
func WithShortDescription(short string) Option {
	return func(a *App) {
		a.Short = short
	}
}

// WithLongDescription sets the application long description.
func WithLongDescription(long string) Option {
	return func(a *App) {
		a.Long = long
	}
}