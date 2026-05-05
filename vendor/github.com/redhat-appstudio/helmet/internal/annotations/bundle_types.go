package annotations

import (
	"fmt"
	"strings"
)

// ParseBundleTypesSupported reports whether a chart may appear under
// installer.integrations (integration bundle) and/or products (full product install).
//
// New annotation helmet.../bundle-types-supported (preferred):
//   - "integration" — integration Secret + validation only (listed under integrations)
//   - "product" — full product only (listed under products; fails if listed under integrations)
//   - "integration,product" or "both" — either placement is valid; runtime mode follows helmet.yaml
//
// Legacy helmet.../bundle-type (used when bundle-types-supported is empty):
//   - (empty) — product-only (same as charts without annotations)
//   - "integration" — integration-only bundle
//   - "product" — product-only
//   - "dual" — both integration and product workflows
func ParseBundleTypesSupported(bundleTypesSupported, legacyBundleType string) (integration, product bool, err error) {
	s := strings.TrimSpace(bundleTypesSupported)
	if s != "" {
		return parseNewBundleTypesSupported(s)
	}
	return parseLegacyBundleType(strings.TrimSpace(legacyBundleType))
}

func parseLegacyBundleType(legacy string) (integration, product bool, err error) {
	switch strings.ToLower(legacy) {
	case "":
		return false, true, nil
	case "integration":
		return true, false, nil
	case "dual":
		return true, true, nil
	case "product":
		return false, true, nil
	default:
		return false, false, fmt.Errorf("unknown legacy %s value %q", BundleType, legacy)
	}
}

func parseNewBundleTypesSupported(s string) (integration, product bool, err error) {
	parts := strings.Split(s, ",")
	for _, p := range parts {
		tok := strings.ToLower(strings.TrimSpace(p))
		if tok == "" {
			continue
		}
		switch tok {
		case "integration":
			integration = true
		case "product":
			product = true
		case "both":
			integration, product = true, true
		default:
			return false, false, fmt.Errorf("unknown %s token %q", BundleTypesSupported, tok)
		}
	}
	if !integration && !product {
		return false, false, fmt.Errorf("%s must list at least one of integration, product, or both", BundleTypesSupported)
	}
	return integration, product, nil
}
