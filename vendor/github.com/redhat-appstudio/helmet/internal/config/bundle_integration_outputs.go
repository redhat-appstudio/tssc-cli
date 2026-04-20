package config

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/redhat-appstudio/helmet/internal/chartfs"
	"github.com/redhat-appstudio/helmet/internal/constants"
)

const (
	distributedBundlesRelDir = "bundles"
	distributedChartsRelDir  = "charts"
)

// OutputTypeIntegrationSecret is the bundle output kind for integration Secrets —
// Secrets downstream charts consume through the integrations framework (names like
// tssc-<integration>-integration), not unrelated operator Secrets.
const OutputTypeIntegrationSecret = "integration_secret"

var integrationSecretFullNamePattern = regexp.MustCompile(
	`^tssc-[a-z0-9](?:[a-z0-9-]*[a-z0-9])?-integration$`,
)

var (
	integrationSecretMetadataNameRX = regexp.MustCompile(
		`(?m)^\s*name:\s*(tssc-[a-z0-9-]+-integration)\s*$`,
	)
	kindSecretLine = regexp.MustCompile(`(?mi)^kind:\s*Secret\s*$`)
	yamlDocSplit   = regexp.MustCompile(`(?m)^---\s*$`)

	fromLiteralKeys = regexp.MustCompile(`--from-literal=(?:"([^"]+)=|([^=]+)=)`)

	argocdEnvKeysRX = regexp.MustCompile(`(?m)^(ARGOCD_[A-Z0-9_]+)=\$\{`)
)

func looksLikeBundledDiscoveryFile(rel string) bool {
	rel = filepath.ToSlash(rel)
	switch {
	case strings.Contains(rel, "/templates/"):
		return strings.HasSuffix(rel, ".yaml") ||
			strings.HasSuffix(rel, ".yml") ||
			strings.HasSuffix(rel, ".tpl")
	case strings.Contains(rel, "/scripts/"):
		return strings.HasSuffix(rel, ".sh")
	default:
		return false
	}
}

func mergeSecretsFromScripts(content string, out map[string][]string) {
	if gn := guessStackroxIntegrationSecret(content); gn != "" {
		from := extractFromLiteralsAcrossScript(content)
		appendKeysMerge(out, gn, from)
	}
	if strings.Contains(content, "argocd-helper.sh") ||
		strings.Contains(content, "argocd_store_credentials") {
		for _, mk := range argocdEnvKeysRX.FindAllStringSubmatch(content, -1) {
			if len(mk) > 1 {
				appendKeysMerge(out, "tssc-argocd-integration", []string{mk[1]})
			}
		}
	}
}

func guessStackroxIntegrationSecret(content string) string {
	if strings.Contains(content, "stackrox-helper.sh") ||
		strings.Contains(content, "StackRox API token") ||
		strings.Contains(content, "store_api_token_in_secret") {
		return "tssc-acs-integration"
	}
	return ""
}

func extractFromLiteralsAcrossScript(content string) []string {
	set := map[string]struct{}{}
	for _, m := range fromLiteralKeys.FindAllStringSubmatch(content, -1) {
		var k string
		switch {
		case len(m) > 1 && m[1] != "":
			k = m[1]
		case len(m) > 2 && m[2] != "":
			k = strings.TrimSpace(m[2])
		default:
			continue
		}
		if k != "" && !strings.ContainsAny(k, "$`\"\t ") {
			set[k] = struct{}{}
		}
	}
	return sortedStrings(set)
}

func sortedStrings(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func appendKeysMerge(dest map[string][]string, secretName string, keys []string) {
	if secretName == "" || len(keys) == 0 {
		return
	}
	have := dest[secretName]
	seen := map[string]struct{}{}
	for _, k := range have {
		seen[k] = struct{}{}
	}
	for _, k := range keys {
		if _, ok := seen[k]; !ok {
			have = append(have, k)
			seen[k] = struct{}{}
		}
	}
	sort.Strings(have)
	dest[secretName] = have
}

func mergeSecretsFromYAMLDocs(content string, dest map[string][]string) {
	blocks := yamlDocSplit.Split(content, -1)
	for _, blk := range blocks {
		raw := strings.TrimSpace(blk)
		if raw == "" || !kindSecretLine.MatchString(raw) {
			continue
		}
		nameMatch := integrationSecretMetadataNameRX.FindStringSubmatch(raw)
		if len(nameMatch) < 2 {
			continue
		}
		appendKeysMerge(dest, nameMatch[1], extractSecretManifestKeys(raw))
	}
}

func extractSecretManifestKeys(secretDoc string) []string {
	lower := strings.ToLower(secretDoc)
	var remainder string
	if i := strings.Index(lower, "\nstringdata:"); i >= 0 {
		remainder = secretDoc[i+1:]
	} else if i := strings.Index(lower, "\ndata:"); i >= 0 {
		remainder = secretDoc[i+1:]
	} else {
		return nil
	}
	keyLine := regexp.MustCompile(`^  ([a-zA-Z0-9_.-]+):\s`)
	set := map[string]struct{}{}
	for _, ln := range strings.Split(remainder, "\n")[1:] {
		if ts := strings.TrimSpace(ln); ts == "" || strings.HasPrefix(ts, "#") {
			continue
		}
		if !strings.HasPrefix(ln, "  ") || len(ln) < 5 {
			break
		}
		if m := keyLine.FindStringSubmatch(ln); m != nil {
			k := m[1]
			if strings.Contains(k, "{{") || strings.Contains(k, "}}") {
				continue
			}
			set[k] = struct{}{}
		}
	}
	return sortedStrings(set)
}

func isMissingPathErr(err error) bool {
	return err != nil && errors.Is(err, fs.ErrNotExist)
}

func scanChartsFSRoot(cfs *chartfs.ChartFS, root string, dest map[string][]string) error {
	err := cfs.WalkDir(filepath.ToSlash(strings.TrimPrefix(root, "/")), func(rel string, ent fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if ent.IsDir() {
			return nil
		}
		if !looksLikeBundledDiscoveryFile(rel) {
			return nil
		}
		body, err := cfs.ReadFile(rel)
		if err != nil {
			return err
		}
		s := string(body)
		mergeSecretsFromYAMLDocs(s, dest)
		mergeSecretsFromScripts(s, dest)
		return nil
	})
	if err != nil && isMissingPathErr(err) {
		return nil
	}
	return err
}

// discoverProductIntegrationSecrets scans bundle charts and legacy charts/<chartName>
// layouts for manifests that declare integration Secret names (`tssc-*-integration`).
func discoverProductIntegrationSecrets(cfs *chartfs.ChartFS, bundleID string, helmChartName string) (map[string][]string, error) {
	dest := map[string][]string{}
	roots := []string{
		path.Join(distributedBundlesRelDir, bundleID, "charts"),
		path.Join(distributedChartsRelDir, helmChartName),
	}
	for _, root := range roots {
		if err := scanChartsFSRoot(cfs, root, dest); err != nil {
			return nil, err
		}
	}
	return dest, nil
}

func stringSlicesEqualIgnoringOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	cp := append([]string(nil), a...)
	cq := append([]string(nil), b...)
	sort.Strings(cp)
	sort.Strings(cq)
	for i := range cp {
		if cp[i] != cq[i] {
			return false
		}
	}
	return true
}

// ValidateProductIntegrationOutputs verifies config.yaml `.outputs` describes only integration
// Secrets (`type` integration_secret) and lists names/keys declared by the product charts.
func ValidateProductIntegrationOutputs(cfs *chartfs.ChartFS, bundleID string, helmChartName string, outputs []BundleOutput) error {
	if len(outputs) == 0 {
		return nil
	}
	discovered, err := discoverProductIntegrationSecrets(cfs, bundleID, helmChartName)
	if err != nil {
		return fmt.Errorf("%s bundles/%s / %s outputs: %w", constants.ConfigFilename, bundleID, helmChartName, err)
	}
	for i := range outputs {
		o := outputs[i]
		if strings.TrimSpace(o.Type) != OutputTypeIntegrationSecret {
			return fmt.Errorf(
				"%s product %s outputs[%d].type=%q invalid: use %q",
				constants.ConfigFilename, bundleID, i, strings.TrimSpace(o.Type), OutputTypeIntegrationSecret)
		}
		name := strings.TrimSpace(o.Name)
		if name == "" {
			return fmt.Errorf("%s product %s outputs[%d]: empty name", constants.ConfigFilename, bundleID, i)
		}
		if !integrationSecretFullNamePattern.MatchString(name) {
			return fmt.Errorf(
				"%s product %s outputs[%d].name=%q must look like tssc-<id>-integration",
				constants.ConfigFilename, bundleID, i, name)
		}
		keysSeen, exists := discovered[name]
		if !exists || len(keysSeen) == 0 {
			return fmt.Errorf(
				"%s product %s outputs[%d]: integration secret %q not found under bundles/%s/charts or %s/%s",
				constants.ConfigFilename, bundleID, i, name, bundleID, distributedChartsRelDir, helmChartName)
		}
		if len(o.Data) > 0 && !stringSlicesEqualIgnoringOrder(keysSeen, o.Data) {
			return fmt.Errorf(
				"%s product %s outputs[%d] secret %q: data keys %#v ≠ discovered %#v",
				constants.ConfigFilename, bundleID, i, name, sortedCopy(o.Data), keysSeen)
		}
	}
	return nil
}

func sortedCopy(s []string) []string {
	cp := append([]string(nil), s...)
	sort.Strings(cp)
	return cp
}
