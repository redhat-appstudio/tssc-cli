Dependency Topology
-------------------

# Dependency Resolution

To enable dynamic dependency resolution annotations are added to the `Chart.yaml` files of TSSC Helm charts. These annotations guide the installer in building the correct deployment order, and the namespaces is determined by the installer [configuration](../README.md#-installerproducts-).

Product associated Helm charts will use the namespace determined in the configuration, while the other charts will use the same namespace as the installer (`.tssc.namespace`).

Use the `topology` subcommand to inspect the order of Helm charts and metadata:

```sh
tssc topology
```

## Annotations

### `tssc.redhat-appstudio.github.com/product-name`

- **Purpose**: This **optional** annotation identifies the product name that a Helm chart is associated with. It links a specific Helm chart to a product defined in `config.yaml` (e.g., `.tssc.products[{name: "OpenShift GitOps"}]`). A single Helm chart should be associated with each product for consistency.
- **Example**: For the OpenShift GitOps product, whose Helm chart is `charts/tssc-gitops`:

```yml
annotations:
  tssc.redhat-appstudio.github.com/product-name: "OpenShift GitOps"
```

### `tssc.redhat-appstudio.github.com/use-product-namespace`

- **Purpose**: This **optional** annotation specifies that a Helm chart, which is *not* directly associated with a product via `product-name`, should be deployed into the namespace of a *specific* product. This is useful for common dependencies that logically belong within a product's deployment but are not the primary product chart. If `product-name` is also present on the chart, the `product-name` annotation takes precedence for namespace determination.
- **Usage**: The value of this annotation must be the name of an enabled product defined in `config.yaml`. The chart will be deployed into the namespace configured for that product.
- **Example**: If `tssc-common-dependency` should be deployed into the `OpenShift GitOps` product's namespace, but is not the `OpenShift GitOps` product chart itself:

```yaml
annotations:
  tssc.redhat-appstudio.github.com/use-product-namespace: "OpenShift GitOps"
```

### `tssc.redhat-appstudio.github.com/depends-on`

- **Purpose**: This **optional** annotation declares direct dependencies of the current chart. It specifies a comma-separated list of chart names that must be successfully deployed *before* this chart.
- **Usage**: The installer uses this annotation to build a topological deployment order, ensuring that all dependencies are met before a chart is deployed.
- **Example**: If the OpenShift GitOps Helm chart depends on `tssc-openshift` and `tssc-subscriptions`:

```yaml
annotations:
  tssc.redhat-appstudio.github.com/depends-on: "tssc-openshift, tssc-subscriptions"
```

## Resolution Logic

The Resolver's core logic for determining the Helm chart deployment order is based on a two-phase process to build a comprehensive deployment topology.

1. **Resolving Enabled Products**: First it iterates through all products enabled in the cluster `config.yaml`. For each enabled product, it identifies its associated Helm chart, appends it to the deployment topology, and then recursively calls depends-on annotation inspection to ensure all of its direct and indirect dependencies are also added to the topology, before the product chart itself.
2. **Resolving Remaining Dependencies**: Then, it performs a final pass over all available Helm charts. It identifies any charts that are not directly associated with a product but are required by other charts. These standalone dependencies are then appended to the topology in their correct order, and their own dependencies are recursively resolved via depends-on inspection. The depends-on inspection ensures that any chart a given chart relies on is placed earlier in the deployment sequence.

# Determine Namespace

The target namespace for each Helm chart will be determined based on the presence of the `product-name` annotation:

- **Product-Associated Charts**: If a Helm chart has the `tssc.redhat-appstudio.github.com/product-name` annotation, it will be deployed into the namespace specified by `.tssc.products[{name: product-name}].namespace` in `config.yaml`.
- **Standalone (Dependency) Charts**: If a Helm chart does *not* have the `tssc.redhat-appstudio.github.com/product-name` annotation (i.e., it's a common dependency), it will be deployed into the installer's default namespace (`.tssc.namespace`) from `config.yaml`.

This approach provides a predictable namespace per Helm chart: typically, products are deployed in their own namespaces, while their common dependencies use the installer's default namespace.
