## Deploying RHADS for development

Once you have a cluster ready:

1. Set up your `.env` file (or `.envrc` or whatever you prefer). Copy the `.env.template` file and fill in the values according to the inline instructions.
2. Source your `.env` file
3. Run `integration-tests/scripts/install.sh`
4. When finished, the script will print a Homepage URL. You can then manually configure webhook URLs and callback URLs in your GitHub or GitLab app settings using the Homepage URL as the base.

## Using install.sh for RHADS Deployment

The `install.sh` script supports flexible deployment configurations through environment variables. You can customize your RHADS deployment by setting these configuration arrays in your `.env` file:

### Configuration Options

#### ACS/TPA Configuration
```bash
# Example: Deploy local ACS instance (default)
export acs_config="local"

# Example: Use remote/existing TPA instance
export tpa_config="remote"
```

**Options**: `local` OR `remote` (single value only)
- `local`: Installs ACS/TPA locally in your cluster
- `remote`: Connects to an existing external ACS/TPA instance


#### Registry Configuration
```bash
# Example: Use multiple registries (comma-separated)
export registry_config="quay,artifactory,nexus"
```

**Options**: `quay`, `artifactory`, `nexus` (can be multiple values)
- `quay`: Integrates with external Quay.io service
- `artifactory`: Integrates with Artifactory registry
- `nexus`: Integrates with Nexus registry
- Multiple values: Comma-separated list to integrate with multiple registries

#### SCM (Source Code Management) Configuration
```bash
# Example: Use multiple SCMs (comma-separated)
export scm_config="github,gitlab"
```

**Options**: `github`, `gitlab`, `bitbucket` (can be multiple values)
- `github`: Integrates with GitHub
- `gitlab`: Integrates with GitLab
- `bitbucket`: Integrates with Bitbucket
- Multiple values: Comma-separated list to integrate with multiple SCM providers

#### Pipeline Configuration
```bash
# Example: Use multiple pipelines (comma-separated)
export pipeline_config="tekton,jenkins"
```

**Options**: `tekton`, `jenkins` (can be multiple values)
- `tekton`: Uses Tekton pipelines (OpenShift Pipelines)
- `jenkins`: Integrates with external Jenkins server
- `azure`: Integrates with Azure DevOps pipelines
- `actions`: Uses GitHub Actions pipelines (requires `github` in SCM config)
- `gitlabci`: Uses GitLab CI pipelines (requires `gitlab` in SCM config)
- Multiple values: Comma-separated list to use the pipeline systems

#### Pre-Release Configuration
```bash
# Example: Configure pre-release subscription for RHDH
export PRE_RELEASE="RHDH"

# Example: Configure pre-release subscription for TAS (auto-detect latest)
export PRE_RELEASE="TAS"
export GITHUB_TOKEN="ghp_xxxxx"  # Required for private repos, recommended for public repos to avoid rate limits

# Example: Configure pre-release subscription for TAS (specific version)
export PRE_RELEASE="TAS"
export TAS_VERSION="1.3.1"
export GITHUB_TOKEN="ghp_xxxxx"  # Required for private repos, recommended for public repos to avoid rate limits

# Example: Configure pre-release subscription for TAS (specific operator version)
export PRE_RELEASE="TAS"
export TAS_VERSION="1.3.2"
export TAS_OPERATOR_VERSION="rhtas-operator.v1.3.2"  # Exact CSV version to install
export GITHUB_TOKEN="ghp_xxxxx"  # Required for private repos, recommended for public repos to avoid rate limits

# Example: Configure pre-release subscription for TAS (specific path - private repo)
export PRE_RELEASE="TAS"
export TAS_RELEASE_PATH="https://github.com/securesign/releases/blob/release-1.3.1/1.3.1/stable"
export GITHUB_TOKEN="ghp_xxxxx"  # Required for private repositories (or use GITOPS_GIT_TOKEN as fallback)

# Example: Configure pre-release subscription for TPA
export PRE_RELEASE="TPA"
export GITHUB_TOKEN="ghp_xxxxx"  # Optional, for private repositories
```

**Options**: `RHDH`, `TPA`, `TAS` (single value only)
- `RHDH`: Configures pre-release subscription for Red Hat Developer Hub
  - Uses the latest release automatically via the RHDH install script
- `TPA`: Configures pre-release subscription for Trusted Profile Analyzer
  - Automatically configures ImageDigestMirrorSet, CatalogSource, and Subscription
  - Uses channel `stable-v1.1` and version `v1.1.1-rc`
- `TAS`: Configures pre-release subscription for Trusted Artifact Signer
  - **Option 1 (Recommended)**: Auto-detect latest release
    - Set `GITHUB_TOKEN` (or `GITOPS_GIT_TOKEN` as fallback)
    - Required for private repositories
    - Recommended for public repositories to avoid GitHub API rate limits
    - The script will automatically fetch the latest release from GitHub
  - **Option 2**: Specify a version
    - Set `TAS_VERSION` (e.g., `1.3.1`)
    - `GITHUB_TOKEN` (or `GITOPS_GIT_TOKEN` as fallback) is required for private repositories
    - Recommended for public repositories to avoid GitHub API rate limits
    - The script will fetch the release matching that version
  - **Option 2b**: Specify exact operator version
    - Set `TAS_VERSION` (e.g., `1.3.2`) and `TAS_OPERATOR_VERSION` (e.g., `rhtas-operator.v1.3.2`)
    - `TAS_OPERATOR_VERSION` overrides the `startingCSV` in the subscription to install a specific operator version
    - `GITHUB_TOKEN` (or `GITOPS_GIT_TOKEN` as fallback) is required for private repositories
    - Recommended for public repositories to avoid GitHub API rate limits
  - **Option 3**: Use a specific path
    - Set `TAS_RELEASE_PATH` to the GitHub release path
    - Example: `export TAS_RELEASE_PATH="https://github.com/securesign/releases/blob/release-1.3.1/1.3.1/stable"`
    - For private repositories: `GITHUB_TOKEN` (or `GITOPS_GIT_TOKEN` as fallback) is required
    - For public repositories: `GITHUB_TOKEN` is not strictly required but recommended to avoid rate limits

**Note**: This parameter is optional. If not set, the pre-release configuration step will be skipped. When set, the script will run `hack/pre-release/pre-release.sh` which dispatches to product-specific scripts (`pre-release-rhdh.sh`, `pre-release-tas.sh`, `pre-release-tpa.sh`) to configure the subscription channels and sources for the specified product before creating the cluster configuration.


Note: 
1. once you've set up your .env for the first time, most of the variables will be re-usable for future deployments.

2. If you are going to use the hosted ACS that we already installed on rhtap-services Cluster, it's already configured the integration with our Artifactory, Nexus servers. 