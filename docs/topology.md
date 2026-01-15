# Dependency Topology

The framework automatically resolves installation order based on chart dependencies, weights, and product associations.

## Viewing the Topology

```sh
<installer-name> topology
```

Output shows installation order, weights, namespaces, and dependencies for each chart.

## Chart Annotations

Add these annotations to `Chart.yaml` to declare dependencies and metadata. All annotations use the `helmet.redhat-appstudio.github.com/` prefix.

### `product-name`

Associates a chart with a product from `config.yaml`. The chart uses the product's configured namespace.

```yaml
annotations:
  helmet.redhat-appstudio.github.com/product-name: "my-product"
```

### `use-product-namespace`

Deploys a dependency chart into a specific product's namespace (for charts without `product-name`).

```yaml
annotations:
  helmet.redhat-appstudio.github.com/use-product-namespace: "my-product"
```

### `depends-on`

Comma-separated list of charts that must be deployed first.

```yaml
annotations:
  helmet.redhat-appstudio.github.com/depends-on: "database, cache"
```

### `weight`

Integer defining installation priority. Higher weights install earlier. Default is 0.

```yaml
annotations:
  helmet.redhat-appstudio.github.com/weight: "500"
```

Common patterns:
- 1000+: Infrastructure (namespaces, operators)
- 500-999: Platform services (databases)
- 100-499: Application services
- 0-99: User applications

### `integrations-provided`

Comma-separated list of integrations this chart creates.

```yaml
annotations:
  helmet.redhat-appstudio.github.com/integrations-provided: "github"
```

### `integrations-required`

CEL expression specifying required integrations. Supports `&&`, `||`, `!`, and parentheses.

```yaml
annotations:
  helmet.redhat-appstudio.github.com/integrations-required: "github && (s3 || azure-storage)"
```

## Resolution Algorithm

**Phase 1**: For each enabled product:
- Find chart with matching `product-name`
- Recursively resolve dependencies via `depends-on`
- Add to topology (dependencies before product)

**Phase 2**: Resolve remaining standalone charts and their dependencies.

**Ordering**: Charts sorted by dependencies first, then by weight (descending).

**Validation**: Detects circular dependencies and validates integration requirements.

## Namespace Assignment

Priority order:
1. Chart with `product-name` → product's namespace from config
2. Chart with `use-product-namespace` → specified product's namespace
3. Otherwise → installer's default namespace

Example:
```yaml
# config.yaml
myinstaller:
  namespace: myapp
  products:
    api:
      namespace: api-ns

# charts/api-server/Chart.yaml
annotations:
  product-name: "api"
# → Deploys to: api-ns

# charts/database/Chart.yaml
annotations:
  use-product-namespace: "api"
# → Deploys to: api-ns

# charts/cache/Chart.yaml
# (no annotations)
# → Deploys to: myapp
```

## Best Practices

- Use high weights (1000+) for infrastructure
- Prefer explicit `depends-on` over implicit weight ordering
- One `product-name` per product
- Validate topology before production: `<installer-name> topology`
- Document complex dependencies in Chart.yaml comments
