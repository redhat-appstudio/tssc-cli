package integration

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/redhat-appstudio/helmet/internal/config"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	corev1 "k8s.io/api/core/v1"
)

// GitLab represents the GitLab integration coordinates.
type GitLab struct {
	logger *slog.Logger // application logger

	insecure  bool   // skip tls verification
	host      string // gitlab host
	port      int    // gitlab port
	group     string // gitlab group name
	appID     string // gitlab application client id
	appSecret string // gitlab application client secret
	token     string // api token credentials
}

var _ Interface = &GitLab{}

// PersistentFlags adds the persistent flags to the informed Cobra command.
func (g *GitLab) PersistentFlags(c *cobra.Command) {
	p := c.PersistentFlags()

	p.StringVar(&g.host, "host", g.host,
		"GitLab hostname")
	p.IntVar(&g.port, "port", g.port,
		"GitLab port")
	p.BoolVar(&g.insecure, "insecure", g.insecure,
		"Skips TLS verification on API calls")
	p.StringVar(&g.group, "group", g.group,
		"GitLab group name")
	p.StringVar(&g.appID, "app-id", g.appID,
		"GitLab application client ID")
	p.StringVar(&g.appSecret, "app-secret", g.appSecret,
		"GitLab application client secret")
	p.StringVar(&g.token, "token", g.token,
		"GitLab API token")

	for _, f := range []string{"token", "group"} {
		if err := c.MarkPersistentFlagRequired(f); err != nil {
			panic(err)
		}
	}
}

// SetArgument sets additional arguments to the integration.
func (g *GitLab) SetArgument(string, string) error {
	return nil
}

// LoggerWith decorates the logger with the integration flags.
func (g *GitLab) LoggerWith(logger *slog.Logger) *slog.Logger {
	return logger.With(
		"host", g.host,
		"port", g.port,
		"insecure", g.insecure,
		"group", g.group,
		"app-id", g.appID,
		"app-secret-len", len(g.appSecret),
		"token-len", len(g.token),
	)
}

// log logger with integration attributes.
func (g *GitLab) log() *slog.Logger {
	return g.LoggerWith(g.logger)
}

// Type returns the type of the integration.
func (g *GitLab) Type() corev1.SecretType {
	return corev1.SecretTypeOpaque
}

// Validate validates the integration configuration.
func (g *GitLab) Validate() error {
	if g.appID != "" && g.appSecret == "" {
		return fmt.Errorf("app-secret is required when id is specified")
	}
	if g.appID == "" && g.appSecret != "" {
		return fmt.Errorf("app-id is required when app-secret is specified")
	}
	return nil
}

// getCurrentGitLabUser returns the current username authenticated, using the
// informed access token.
func (g *GitLab) getCurrentGitLabUser() (string, error) {
	gitLabURL := fmt.Sprintf("https://%s", g.host)
	if g.port != 443 {
		gitLabURL += fmt.Sprintf(":%d", g.port)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: g.insecure,
			MinVersion:         tls.VersionTLS12,
		},
	}

	client, err := gitlab.NewClient(
		g.token,
		gitlab.WithBaseURL(gitLabURL),
		gitlab.WithHTTPClient(&http.Client{Transport: transport}),
	)
	if err != nil {
		g.log().Error("Error building gitlab client")
		return "", err
	}

	user, _, err := client.Users.CurrentUser()
	if err != nil {
		g.log().Error("Error getting user")
		return "", err
	}
	return user.Username, nil
}

// Data returns the GitLab integration data, using the local configuration and
// username obtained on the fly.
func (g *GitLab) Data(
	_ context.Context,
	_ *config.Config,
) (map[string][]byte, error) {
	username, err := g.getCurrentGitLabUser()
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		"host":         []byte(g.host),
		"port":         []byte(strconv.Itoa(g.port)),
		"group":        []byte(g.group),
		"clientId":     []byte(g.appID),
		"clientSecret": []byte(g.appSecret),
		"username":     []byte(username),
		"token":        []byte(g.token),
	}, nil
}

// NewGitLab instantiate a new GitLab integration. By default it uses the public
// GitLab host.
func NewGitLab(logger *slog.Logger) *GitLab {
	return &GitLab{
		logger: logger,
		host:   "gitlab.com",
		port:   443,
	}
}
