<p align="center">
    <a alt="Project quality report" href="https://goreportcard.com/report/github.com/redhat-appstudio/rhtap-cli">
        <img src="https://goreportcard.com/badge/github.com/redhat-appstudio/rhtap-cli">
    </a>
    <a alt="Release workflow status" href="https://github.com/redhat-appstudio/rhtap-cli/actions">
        <img src="https://github.com/redhat-appstudio/rhtap-cli/actions/workflows/release.yaml/badge.svg">
    </a>
    <a alt="Latest project release" href="https://github.com/redhat-appstudio/rhtap-cli/releases/latest">
        <img src="https://img.shields.io/github/v/release/redhat-appstudio/rhtap-cli">
    </a>
</p>

Red Hat Trusted Application Pipeline Installer (`rhtap-cli`)
------------------------------------------------------------

# Abstract

The `rhtap-cli` is designed as a sophisticated installer for Kubernetes [Helm Charts][helm], addressing the complexity of managing interdependent resources in Kubernetes environments. Unlike Kubernetes, which orchestrates resources individually without acknowledging their interdependencies, `rhtap-cli` enhances the deployment process by considering these relationships, thereby improving the user experience.

This CLI leverages a [`config.yaml`](installer/config.yaml) file to sequence Helm Chart deployments meticulously. It ensures the integrity of each deployment phase by executing a comprehensive test suite before proceeding to the next Chart installation. This methodical approach guarantees that each phase is successfully completed, enhancing reliability and stability.

Helm, serving as the foundation of `rhtap-cli`, provides a detailed blueprint of resources within Kubernetes. This allows for thorough inspection and troubleshooting of deployment issues, offering users detailed documentation and tips for resolution. By integrating with Helm Charts, `rhtap-cli` not only adheres to industry standards but also opens the door to more sophisticated features, further enriching the deployment experience.

The `rhtap-cli` is designed to be user-friendly, providing a seamless installation process for users of all skill levels. 

# Deploy RHTAP

Install the `rhtap-cli` binary on your local machine following [these instructions](#installing-rhtap-cli).

Follow the below steps to deploy RHTAP on Openshift cluster. 

1. Edit the [`config.yaml`](installer/config.yaml) file for select or deselect the components from installation. See the [configuration](#configuration) section for more details.

  For instance: change the lines as below to disable installation of components ACS and Quay .

```yaml
# ...
redHatAdvancedClusterSecurity: 
  enabled: false 
redHatQuay: 
  enabled: false
```
      
2. Run the command `rhtap-cli` to display help text that shows all the supported commands and options. 

3. Run the command `rhtap-cli integration` to provide integrations to external components. The command below lists the options supported: 
  
```bash
rhtap-cli integration --help
```
  
4. Finally, run the below command to proceed with RHTAP deployment. 

```bash
rhtap-cli deploy
```

# Configuration

The [`config.yaml`](installer/config.yaml) file is structured to outline key components essential for the setup, for instance:

```yaml
---
rhtapCLI:
  namespace: rhtap
  features: {}
  dependencies: {}
```

The attributes of the `rhtapCLI` object are as follows:

- `.namespace`: Specifies the default namespace used by the installer, set to rhtap. This namespace acts as the primary operational area for the installation process.
- `.features`: Defines the features to be deployed by the installer. Each feature is identified by a unique name and a set of properties.
- `.dependencies`: Specifies the dependencies rolled out by the installer in the specific order defined in the configuration file.

## `rhtapCLI.features`

Defines the features the installer will deploy. Each feature is defined by a unique name and a set of properties. For instance, the following snippet defines a `featureName` block:

```yaml
---
rhtapCLI:
  features:
    featureName:
      enabled: true
      namespace: namespace
      properties:
        key: value
```

With the following attributes:
- `enabled`: A boolean value to toggle the unique feature
- `namespace`: The namespace in which the feature will be deployed
- `properties`: A set of key-value pairs to define the feature's properties

This data can be leveraged for templating using the [`values.yaml.tpl`](#template-functions) file.

## `rhtapCLI.dependencies`

Each dependency is defined by a unique name and a set of attributes. The installer will deploy these dependencies in the order specified in the configuration file. For instance:

```yaml
rhtapCLI:
  dependencies:
    - chart: path/to/chart/directory 
      namespace: namespace
      enabled: true
```

### Hook Scripts

The installer supports hook scripts to execute custom logic before and after the installation of a Helm Chart. The hook scripts are stored in the `hooks` directory and are executed in the following order:

1. `pre-install.sh`: Executed before the installation of the dependency.
2. `post-install.sh`: Executed after the installation of the dependency.

Windows users must be aware that the hook scripts are written in Bash and may not be compatible with the Windows shell. To execute the hook scripts, consider using WSL or a similar tool.

## Template Functions

The following functions are available for use in the [`values.yaml.tpl`](./installer/charts/values.yaml.tpl) file:

### `{{ .Installer.Features.* }}`

- `{{ .Installer.Features.*.Enabled }}`: Returns the boolean value of the feature's `enabled` field.
- `{{ .Installer.Features.*.Namespace }}`: Returns the namespace in which the feature will be deployed.
- `{{ .Installer.Features.*.Properties.*}}`: Returns a dictionary of key-value pairs for the feature's properties.

### `{{ .OpenShift.Ingress }}`

Helper function to inspect the target cluster's Ingress configuration.

```yaml
{{- $ingressDomain := required "OpenShift ingress domain" .OpenShift.Ingress.Domain -}}
---
developerHub:
  ingressDomain: {{ $ingressDomain }}
```

# Installing `rhtap-cli`

## Pre-Compiled Binaries

Check the lastest release from the [releases page][releases] and download the binary for your operating system and executable architecture. Then, either use the binary directly or move it to a directory in your `PATH`, for instance:

```bash
install --mode=755 bin/rhtap-cli /usr/local/bin
```

## From Source

Please refer to the [CONTRIBUTING.md](CONTRIBUTING.md) for more information on building the project from source requirements. Then, follow the steps below to install the `rhtap-cli` binary from source:

1. Clone [the repository][rhtapCLI], and navigate to the `rhtap-cli` directory.

```bash
git clone --depth=1 https://github.com/redhat-appstudio/rhtap-cli.git && \
  cd rhtap-cli
```

2. Run the command `make` from the `rhtap-cli` directory, this will create a `bin` folder

```bash
make
```

3. Move the `rhtap-cli` to the desired location, for instance `/usr/local/bin`.

```bash
install --mode=755 bin/rhtap-cli /usr/local/bin
```

# Contributing

Please refer to the [CONTRIBUTING.md](CONTRIBUTING.md) file for more information on contributing to this project.
 
[helm]: https://helm.sh/
[releases]: https://github.com/redhat-appstudio/rhtap-cli/releases
[rhtapCLI]: https://github.com/redhat-appstudio/rhtap-cli
