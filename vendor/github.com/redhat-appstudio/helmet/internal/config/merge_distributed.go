package config

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/redhat-appstudio/helmet/internal/annotations"
	"github.com/redhat-appstudio/helmet/internal/chartfs"
	"github.com/redhat-appstudio/helmet/internal/constants"

	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chartutil"
)

const (
	distributedSettingsPath  = "config/settings.yaml"
	distributedBlueprintPath = "helmet.yaml"
	distributedChartsDir     = "charts"
	distributedBundlesDir    = "bundles"
	localRefPrefix           = "local://"
)

// helmInstallerBlueprint mirrors the installer helmet.yaml blueprint.
type helmInstallerBlueprint struct {
	Name     string   `yaml:"name"`
	Products []string `yaml:"products"`
}

// MergeDistributedInstallerYAML builds one config.yaml payload from settings,
// helmet blueprint, and per-bundle or per-chart config.yaml fragments.
func MergeDistributedInstallerYAML(cfs *chartfs.ChartFS, appIdentifier string) ([]byte, error) {
	settingsBytes, err := cfs.ReadFile(distributedSettingsPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", distributedSettingsPath, err)
	}
	helmBytes, err := cfs.ReadFile(distributedBlueprintPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", distributedBlueprintPath, err)
	}

	var blueprint helmInstallerBlueprint
	if err := yaml.Unmarshal(helmBytes, &blueprint); err != nil {
		return nil, fmt.Errorf("parse helmet blueprint: %w", err)
	}

	if err := validateProductBundleTypes(cfs, blueprint.Products); err != nil {
		return nil, err
	}

	var settings map[string]interface{}
	if err := yaml.Unmarshal(settingsBytes, &settings); err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	}

	var products []Product
	for _, ref := range blueprint.Products {
		id, err := integrationIDFromLocalRef(ref)
		if err != nil {
			return nil, err
		}
		chartName, err := chartNameFromLocalRef(ref)
		if err != nil {
			return nil, err
		}
		cfgPath := path.Join(distributedBundlesDir, id, constants.ConfigFilename)
		body, err := cfs.ReadFile(cfgPath)
		if err != nil {
			cfgPath = path.Join(distributedChartsDir, chartName, constants.ConfigFilename)
			body, err = cfs.ReadFile(cfgPath)
			if err != nil {
				return nil, fmt.Errorf(
					"read product config (try bundles/%s/%s or charts/%s/%s): %w",
					id, constants.ConfigFilename, chartName, constants.ConfigFilename, err)
			}
		}
		var p Product
		if err := yaml.Unmarshal(body, &p); err != nil {
			return nil, fmt.Errorf("parse product config %s: %w", cfgPath, err)
		}
		if err := ValidateProductIntegrationOutputs(cfs, id, chartName, p.Outputs); err != nil {
			return nil, err
		}
		stripped := p
		stripped.Outputs = nil
		products = append(products, stripped)
	}

	doc := map[string]interface{}{
		appIdentifier: map[string]interface{}{
			"settings": settings,
			"products":   products,
		},
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return append([]byte("---\n"), out...), nil
}

func chartNameFromLocalRef(ref string) (string, error) {
	id, err := integrationIDFromLocalRef(ref)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("tssc-%s", id), nil
}

func integrationIDFromLocalRef(ref string) (string, error) {
	if !strings.HasPrefix(ref, localRefPrefix) {
		return "", fmt.Errorf("expected %q reference, got %q", localRefPrefix, ref)
	}
	return strings.TrimPrefix(ref, localRefPrefix), nil
}

// mapAtDotPath returns the mapping at the given dot-separated path (e.g. "quay"
// or "tpa.trustedProfileAnalyzer.integrationSecret").
func mapAtDotPath(root map[string]interface{}, dotPath string) (map[string]interface{}, error) {
	if dotPath == "" {
		return nil, fmt.Errorf("empty values path")
	}
	parts := strings.Split(dotPath, ".")
	var cur interface{} = root
	for i, seg := range parts {
		if seg == "" {
			continue
		}
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("segment %q is not a mapping", strings.Join(parts[:i], "."))
		}
		cur, ok = m[seg]
		if !ok {
			return nil, fmt.Errorf("missing key %q", seg)
		}
	}
	out, ok := cur.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("path must resolve to a mapping, got %T", cur)
	}
	return out, nil
}

func readHelmChartYAML(cfs *chartfs.ChartFS, chartName string) (ann map[string]string, description, nameField string, err error) {
	dir, err := cfs.FindChartDir(chartName)
	if err != nil {
		return nil, "", "", err
	}
	chartPath := path.Join(dir, chartutil.ChartfileName)
	body, err := cfs.ReadFile(chartPath)
	if err != nil {
		return nil, "", "", err
	}
	return parseHelmChartYAML(body)
}

func parseHelmChartYAML(body []byte) (ann map[string]string, description, nameField string, err error) {
	var doc struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Metadata    struct {
			Annotations map[string]string `yaml:"annotations"`
		} `yaml:"metadata"`
		Annotations map[string]string `yaml:"annotations"`
	}
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return nil, "", "", err
	}
	ann = map[string]string{}
	for k, v := range doc.Metadata.Annotations {
		ann[k] = v
	}
	for k, v := range doc.Annotations {
		ann[k] = v
	}
	return ann, doc.Description, doc.Name, nil
}

// validateProductBundleTypes rejects charts that do not support a product bundle
// when listed under products (e.g. integration-only charts).
func validateProductBundleTypes(cfs *chartfs.ChartFS, productRefs []string) error {
	for _, ref := range productRefs {
		chartName, err := chartNameFromLocalRef(ref)
		if err != nil {
			return err
		}
		_, supportsProduct, err := readChartBundleSupport(cfs, chartName)
		if err != nil {
			return fmt.Errorf("read chart %s: %w", chartName, err)
		}
		if !supportsProduct {
			return fmt.Errorf(
				"helmet.yaml: product %q (chart %s) does not support a product bundle; set %s to include product",
				ref, chartName, annotations.BundleTypesSupported)
		}
	}
	return nil
}

func readChartBundleSupport(cfs *chartfs.ChartFS, chartName string) (integration, product bool, err error) {
	ann, _, _, err := readHelmChartYAML(cfs, chartName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, true, nil
		}
		return false, false, err
	}
	return annotations.ParseBundleTypesSupported(
		ann[annotations.BundleTypesSupported],
		ann[annotations.BundleType],
	)
}
