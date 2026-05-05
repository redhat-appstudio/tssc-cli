package config

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"strings"

	"github.com/redhat-appstudio/helmet/internal/annotations"
	"github.com/redhat-appstudio/helmet/internal/chartfs"

	"gopkg.in/yaml.v3"
)

const (
	distributedSettingsPath  = "config/settings.yaml"
	distributedBlueprintPath = "helmet.yaml"
	distributedChartsDir     = "charts"
	localRefPrefix        = "local://"
	chartValuesYAMLFile   = "values.yaml"
)

// helmInstallerBlueprint mirrors the installer helmet.yaml blueprint.
type helmInstallerBlueprint struct {
	Name         string   `yaml:"name"`
	Products     []string `yaml:"products"`
	Integrations []string `yaml:"integrations"`
}

// MergeDistributedInstallerYAML builds one config.yaml payload from settings,
// helmet blueprint, per-chart config.yaml files, and per-chart integration
// metadata (Chart.yaml + values.yaml). Integration property defaults are read from
// values.yaml at the top-level key matching the integration id (helmet.yaml
// local:// suffix, same as charts/tssc-<id>).
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

	if err := validateNoChartInProductsAndIntegrations(
		blueprint.Products, blueprint.Integrations,
	); err != nil {
		return nil, err
	}

	if err := validateProductBundleTypes(cfs, blueprint.Products); err != nil {
		return nil, err
	}
	if err := validateIntegrationBundleTypes(cfs, blueprint.Integrations); err != nil {
		return nil, err
	}

	var settings map[string]interface{}
	if err := yaml.Unmarshal(settingsBytes, &settings); err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	}

	var products []Product
	for _, ref := range blueprint.Products {
		chartName, err := chartNameFromLocalRef(ref)
		if err != nil {
			return nil, err
		}
		cfgPath := path.Join(distributedChartsDir, chartName, "config.yaml")
		body, err := cfs.ReadFile(cfgPath)
		if err != nil {
			return nil, fmt.Errorf("read product config %s: %w", cfgPath, err)
		}
		var p Product
		if err := yaml.Unmarshal(body, &p); err != nil {
			return nil, fmt.Errorf("parse product config %s: %w", cfgPath, err)
		}
		products = append(products, p)
	}

	var integrations []IntegrationSpec
	for _, ref := range blueprint.Integrations {
		entry, err := loadDistributedIntegrationSpec(cfs, ref)
		if err != nil {
			return nil, err
		}
		integrations = append(integrations, entry)
	}

	doc := map[string]interface{}{
		appIdentifier: map[string]interface{}{
			"settings":     settings,
			"integrations": integrations,
			"products":     products,
		},
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return append([]byte("---\n"), out...), nil
}

// validateNoChartInProductsAndIntegrations rejects the same local chart reference
// appearing under both products and integrations (e.g. TPA cannot be both).
func validateNoChartInProductsAndIntegrations(productRefs, integrationRefs []string) error {
	ids := make(map[string]struct{})
	for _, ref := range productRefs {
		id, err := integrationIDFromLocalRef(ref)
		if err != nil {
			return err
		}
		ids[id] = struct{}{}
	}
	for _, ref := range integrationRefs {
		id, err := integrationIDFromLocalRef(ref)
		if err != nil {
			return err
		}
		if _, dup := ids[id]; dup {
			return fmt.Errorf(
				"helmet.yaml: local://%s cannot be listed under both products and integrations; choose one",
				id)
		}
	}
	return nil
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

// loadDistributedIntegrationSpec builds an IntegrationSpec from charts/<tssc-id>/
// Chart.yaml (ids and labels) and values.yaml (default properties for merged config).
// Template-rendered cluster values still override via values.yaml.tpl from
// Installer.Integrations.
func loadDistributedIntegrationSpec(cfs *chartfs.ChartFS, ref string) (IntegrationSpec, error) {
	chartName, err := chartNameFromLocalRef(ref)
	if err != nil {
		return IntegrationSpec{}, err
	}
	installerID, err := integrationIDFromLocalRef(ref)
	if err != nil {
		return IntegrationSpec{}, err
	}
	ann, description, nameField, err := readHelmChartYAML(cfs, chartName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return IntegrationSpec{}, fmt.Errorf(
				"integration %q: chart directory charts/%s not found (add chart or fix helmet.yaml)",
				ref, chartName)
		}
		return IntegrationSpec{}, err
	}
	nameField = strings.TrimSpace(nameField)
	if nameField == "" {
		return IntegrationSpec{}, fmt.Errorf(
			"integration %q: Chart.yaml must set name to %q",
			ref, chartName)
	}
	if nameField != chartName {
		return IntegrationSpec{}, fmt.Errorf(
			"integration %q: Chart.yaml name %q must match chart directory %q",
			ref, nameField, chartName)
	}
	if docID := strings.TrimSpace(ann[annotations.InstallerIntegrationID]); docID != "" && docID != installerID {
		return IntegrationSpec{}, fmt.Errorf(
			"integration %q: %s is %q but must match local ref id %q",
			ref, annotations.InstallerIntegrationID, docID, installerID)
	}
	provided := firstCSVToken(ann[annotations.IntegrationsProvided])
	if provided == "" {
		return IntegrationSpec{}, fmt.Errorf(
			"integration %q: chart %q must set annotation %s (topology / CEL integration name)",
			ref, chartName, annotations.IntegrationsProvided)
	}
	displayName := strings.TrimSpace(ann[annotations.IntegrationDisplayName])
	if displayName == "" {
		displayName = strings.TrimSpace(description)
	}
	if displayName == "" {
		displayName = nameField
	}
	if displayName == "" {
		displayName = chartName
	}
	propsPath := installerID
	valuesRoot, err := readChartValuesYAML(cfs, chartName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return IntegrationSpec{}, fmt.Errorf(
				"integration %q: chart %s/%s not found (needed for integration property defaults)",
				ref, chartName, chartValuesYAMLFile)
		}
		return IntegrationSpec{}, err
	}
	propsMap, err := mapAtDotPath(valuesRoot, propsPath)
	if err != nil {
		return IntegrationSpec{}, fmt.Errorf(
			"integration %q: values.yaml must contain a top-level mapping %q (installer integration id; must match %s suffix): %w",
			ref, propsPath, localRefPrefix, err)
	}
	props := maps.Clone(propsMap)
	return IntegrationSpec{
		ID:         installerID,
		Name:       displayName,
		Properties: props,
	}, nil
}

func readChartValuesYAML(cfs *chartfs.ChartFS, chartName string) (map[string]interface{}, error) {
	p := path.Join(distributedChartsDir, chartName, chartValuesYAMLFile)
	body, err := cfs.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var root map[string]interface{}
	if err := yaml.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("parse %s: %w", p, err)
	}
	if root == nil {
		root = map[string]interface{}{}
	}
	return root, nil
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

func firstCSVToken(s string) string {
	for _, p := range strings.Split(s, ",") {
		t := strings.TrimSpace(p)
		if t != "" {
			return t
		}
	}
	return ""
}

func readHelmChartYAML(cfs *chartfs.ChartFS, chartName string) (ann map[string]string, description, nameField string, err error) {
	chartPath := path.Join(distributedChartsDir, chartName, "Chart.yaml")
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
				"helmet.yaml: product %q (chart %s) does not support a product bundle; list it under integrations only or set %s to include product",
				ref, chartName, annotations.BundleTypesSupported)
		}
	}
	return nil
}

// validateIntegrationBundleTypes rejects charts that do not support an integration
// bundle when listed under integrations (e.g. product-only charts).
func validateIntegrationBundleTypes(cfs *chartfs.ChartFS, integrationRefs []string) error {
	for _, ref := range integrationRefs {
		chartName, err := chartNameFromLocalRef(ref)
		if err != nil {
			return err
		}
		supportsIntegration, _, err := readChartBundleSupport(cfs, chartName)
		if err != nil {
			return fmt.Errorf("read chart %s: %w", chartName, err)
		}
		if !supportsIntegration {
			return fmt.Errorf(
				"helmet.yaml: integration %q (chart %s) does not support an integration bundle; list it under products only or set %s to include integration",
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
