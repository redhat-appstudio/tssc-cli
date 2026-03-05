#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o pipefail

# Function to determine which tssc binary to use
# /usr/local/tssc-bin/tssc is the binary that is used in the CI pipeline
get_tssc_binary() {
  if [ -x "/usr/local/tssc-bin/tssc" ]; then
    echo "/usr/local/tssc-bin/tssc"
  else
    echo "./bin/tssc"
  fi
}

TSSC_BINARY=$(get_tssc_binary)

## This file should be present only in CI created by integration-tests/scripts/ci-oc-login.sh
if [ -f "$HOME/rhtap-cli-ci-kubeconfig" ]; then
    export KUBECONFIG="$HOME/rhtap-cli-ci-kubeconfig"
fi

echo "[INFO]Configuring deployment"
if [[ -n "${acs_config:-}" ]]; then
    # Convert comma-separated values to space-separated, then read into array
    IFS=',' read -ra acs_config <<< "${acs_config}"
else
    acs_config=(local)
fi

if [[ -n "${tpa_config:-}" ]]; then
    IFS=',' read -ra tpa_config <<< "${tpa_config}"
else
    tpa_config=(local)
fi

if [[ -n "${tas_config:-}" ]]; then
    IFS=',' read -ra tas_config <<< "${tas_config}"
else
    tas_config=(local)
fi

if [[ -n "${registry_config:-}" ]]; then
    IFS=',' read -ra registry_config <<< "${registry_config}"
else
    registry_config=(quay)
fi

if [[ -n "${scm_config:-}" ]]; then
    IFS=',' read -ra scm_config <<< "${scm_config}"
else
    scm_config=(github)
fi

if [[ -n "${pipeline_config:-}" ]]; then
    IFS=',' read -ra pipeline_config <<< "${pipeline_config}"
else
    pipeline_config=(tekton)
fi

if [[ -n "${auth_config:-}" ]]; then
    IFS=',' read -ra auth_config <<< "${auth_config}"
else
    auth_config=(github)
fi

# Export after setting
export acs_config tpa_config tas_config registry_config scm_config pipeline_config auth_config

echo "[INFO] acs_config=(${acs_config[*]})"
echo "[INFO] tpa_config=(${tpa_config[*]})"
echo "[INFO] tas_config=(${tas_config[*]})"
echo "[INFO] registry_config=(${registry_config[*]})"
echo "[INFO] scm_config=(${scm_config[*]})"
echo "[INFO] pipeline_config=(${pipeline_config[*]})"
echo "[INFO] auth_config=(${auth_config[*]})"

tpl_file="installer/values.yaml.tpl"
config_file="installer/config.yaml"

ci_enabled() {
  echo "[INFO] Turn ci to true, this is required when you perform rhtap-e2e automation test against TSSC"
  yq -i '.tssc.settings.ci.debug = true' "${config_file}"
}

update_dh_catalog_url() {
  # if DEVELOPER_HUB_CATALOG_URL is not empty string, then update the catalog url
  if [[ -n "${DEVELOPER_HUB_CATALOG_URL}" ]]; then
    echo "[INFO] Update dh catalog url with $DEVELOPER_HUB_CATALOG_URL"
    yq -i '.tssc.products[] |= select(.name == "Developer Hub").properties.catalogURL=strenv(DEVELOPER_HUB_CATALOG_URL)' "${config_file}"
  fi
}

update_dh_auth_config() {
  # Use auth_config to determine the auth provider for Developer Hub
  if [[ " ${auth_config[*]} " =~ " gitlab " ]]; then
    echo "[INFO] Change Developer Hub auth to gitlab"
    yq -i '.tssc.products[] |= select(.name == "Developer Hub").properties.authProvider = "gitlab"' "${config_file}"
  elif [[ " ${auth_config[*]} " =~ " github " ]]; then
    echo "[INFO] Change Developer Hub auth to github"
    yq -i '.tssc.products[] |= select(.name == "Developer Hub").properties.authProvider = "github"' "${config_file}"
  else
    echo "[INFO] Keep Developer Hub auth as oidc (default)"
  fi
}

update_dh_namespace_prefixes() {
  # Update Developer Hub namespace prefixes from TSSC_APP_DEPLOYMENT_NAMESPACES
  if [[ -n "${TSSC_APP_DEPLOYMENT_NAMESPACES:-}" ]]; then
    echo "[INFO] Update Developer Hub namespace prefixes with: $TSSC_APP_DEPLOYMENT_NAMESPACES"

    # Convert comma-separated values to space-separated, then read into array
    IFS=',' read -ra namespace_prefixes <<< "${TSSC_APP_DEPLOYMENT_NAMESPACES}"

    # Clear existing namespacePrefixes array first
    yq -i '.tssc.products[] |= select(.name == "Developer Hub").properties.namespacePrefixes = []' "${config_file}"

    # Add each namespace prefix to the array
    for namespace in "${namespace_prefixes[@]}"; do
      namespace=$(echo "$namespace" | xargs)
      # Skip empty strings
      if [[ -z "$namespace" ]]; then
        continue
      fi
      echo "[INFO] Adding namespace prefix: $namespace"
      yq -i '.tssc.products[] |= select(.name == "Developer Hub").properties.namespacePrefixes += ["'"$namespace"'"]' "${config_file}"
    done
  else
    echo "[INFO] TSSC_APP_DEPLOYMENT_NAMESPACES not set, keeping default namespace prefixes"
  fi
}

# Workaround: This function has to be called before tssc import "installer/config.yaml" into cluster.
# Currently, the `tssc integration github` subcommand lacks the ability to create a secret for an existing application.
github_integration() {
  # Check if "github" is in scm_config array
  # Check if GitHub is as auth_config
  if [[ " ${scm_config[*]} " =~ " github " ]] || [[ " ${auth_config[*]} " =~ " github " ]]; then
    echo "[INFO] Config Github integration with TSSC"

    GITHUB_APP_ID="${GITHUB_APP_ID:-$(cat /usr/local/rhtap-cli-install/rhdh-github-app-id)}"
    GITHUB_APP_CLIENT_ID="${GITHUB_APP_CLIENT_ID:-$(cat /usr/local/rhtap-cli-install/rhdh-github-client-id)}"
    GITHUB_APP_CLIENT_SECRET="${GITHUB_APP_CLIENT_SECRET:-$(cat /usr/local/rhtap-cli-install/rhdh-github-client-secret)}"
    GITHUB_APP_PRIVATE_KEY="${GITHUB_APP_PRIVATE_KEY:-$(base64 -d < /usr/local/rhtap-cli-install/rhdh-github-private-key | sed 's/^/        /')}"
    GITOPS_GIT_TOKEN="${GITOPS_GIT_TOKEN:-$(cat /usr/local/rhtap-cli-install/github_token)}"
    GITHUB_APP_WEBHOOK_SECRET="${GITHUB_APP_WEBHOOK_SECRET:-$(cat /usr/local/rhtap-cli-install/rhdh-github-webhook-secret)}"

    cat << EOF | kubectl apply -f -
kind: Secret
type: Opaque
apiVersion: v1
metadata:
    name: tssc-github-integration
    namespace: tssc
stringData:
    id: "$GITHUB_APP_ID"
    clientId: "$GITHUB_APP_CLIENT_ID"
    clientSecret: "$GITHUB_APP_CLIENT_SECRET"
    host: github.com
    pem: |
$(printf "%s\n" "${GITHUB_APP_PRIVATE_KEY}" | sed 's/^/        /')
    token: "$GITOPS_GIT_TOKEN"
    username: "$GITHUB_AUTH_USERNAME"
    webhookSecret: "$GITHUB_APP_WEBHOOK_SECRET"
EOF
  fi
}

jenkins_integration() {
  if [[ " ${pipeline_config[*]} " =~ " jenkins " ]]; then
    echo "[INFO] Integrates an exising Jenkins server into TSSC"

    JENKINS_API_TOKEN="${JENKINS_API_TOKEN:-$(cat /usr/local/rhtap-cli-install/jenkins-api-token)}"
    JENKINS_URL="${JENKINS_URL:-$(cat /usr/local/rhtap-cli-install/jenkins-url)}"
    JENKINS_USERNAME="${JENKINS_USERNAME:-$(cat /usr/local/rhtap-cli-install/jenkins-username)}"

    "${TSSC_BINARY}" integration --kube-config "$KUBECONFIG" jenkins --token="$JENKINS_API_TOKEN" --url="$JENKINS_URL" --username="$JENKINS_USERNAME" --force
  fi
}

azure_integration() {
  if [[ " ${pipeline_config[*]} " =~ " azure " ]]; then
    echo "[INFO] Integrates an exising Azure DevOps server into TSSC"

    AZURE_TOKEN="${AZURE_TOKEN:-$(cat /usr/local/rhtap-cli-install/azure-token)}"
    AZURE_HOST="${AZURE_HOST:-$(cat /usr/local/rhtap-cli-install/azure-host)}"
    AZURE_ORGANIZATION="${AZURE_ORGANIZATION:-$(cat /usr/local/rhtap-cli-install/azure-organization)}"

    "${TSSC_BINARY}" integration --kube-config "$KUBECONFIG" azure --token="$AZURE_TOKEN" --host="$AZURE_HOST" --organization="$AZURE_ORGANIZATION" --force
  fi
}

gitlab_integration() {
  if [[ " ${scm_config[*]} " =~ " gitlab " ]] || [[ " ${auth_config[*]} " =~ " gitlab " ]]; then
    echo "[INFO] Configure Gitlab integration into TSSC"

    GITLAB__TOKEN="${GITLAB__TOKEN:-$(cat /usr/local/rhtap-cli-install/gitlab_token)}"

    GITLAB__APP__ID="${GITLAB__APP__ID:-$(cat /usr/local/rhtap-cli-install/gitlab-app-id)}"
    GITLAB__APP_SECRET="${GITLAB__APP_SECRET:-$(cat /usr/local/rhtap-cli-install/gitlab-app-secret)}"
    GITLAB__GROUP="${GITLAB__GROUP:-$(cat /usr/local/rhtap-cli-install/gitlab-group)}"

    "${TSSC_BINARY}" integration --kube-config "$KUBECONFIG" gitlab --token="${GITLAB__TOKEN}" --app-id="${GITLAB__APP__ID}" --app-secret="${GITLAB__APP_SECRET}" --group="${GITLAB__GROUP}" --force
  fi
}

quay_integration() {
  if [[ " ${registry_config[*]} " =~  quay ]]; then
    echo "[INFO] Configure quay integration into TSSC"

    QUAY__DOCKERCONFIGJSON="${QUAY__DOCKERCONFIGJSON:-$(cat /usr/local/rhtap-cli-install/quay-dockerconfig-json)}"
    QUAY__API_TOKEN="${QUAY__API_TOKEN:-$(cat /usr/local/rhtap-cli-install/quay-api-token)}"

    "${TSSC_BINARY}" integration --kube-config "$KUBECONFIG" quay --url="https://quay.io" --dockerconfigjson="${QUAY__DOCKERCONFIGJSON}" --token="${QUAY__API_TOKEN}" --force
  fi
}

# Workaround: This function has to be called before tssc import "installer/config.yaml" into cluster.
# Currently, the tssc `config` subcommand lacks the ability to modify property values stored in cluster
disable_acs() {
  # if "remote" is in acs_config array, then disable ACS installation
  # Update the YAML anchor &rhacsEnabled from true to false (line 31 in config.yaml)
  if [[ " ${acs_config[*]} " =~ " remote " ]]; then
    echo "[INFO] Disable ACS installation in the TSSC configuration"
    yq -i '.tssc.products[] |= select(.name == "Advanced Cluster Security").enabled = false' "${config_file}"
  else
    echo "[INFO] ACS is set to local, keeping &rhacsEnabled anchor as true"
  fi
}

acs_integration() {
  if [[ " ${acs_config[*]} " =~ " remote " ]]; then
    echo "[INFO] Configure an existing intance of ACS integration into TSSC"

    ACS_CENTRAL_ENDPOINT="${ACS_CENTRAL_ENDPOINT:-$(cat /usr/local/rhtap-cli-install/acs-central-endpoint)}"
    ACS_API_TOKEN="${ACS_API_TOKEN:-$(cat /usr/local/rhtap-cli-install/acs-api-token)}"

    "${TSSC_BINARY}" integration --kube-config "$KUBECONFIG" acs --endpoint="${ACS_CENTRAL_ENDPOINT}" --token="${ACS_API_TOKEN}" --force 
  fi
}

bitbucket_integration() {
  if [[ " ${scm_config[*]} " =~ " bitbucket " ]]; then
    echo "[INFO] Configure Bitbucket integration into TSSC"

    BITBUCKET_USERNAME="${BITBUCKET_USERNAME:-$(cat /usr/local/rhtap-cli-install/bitbucket-username)}"
    BITBUCKET_APP_PASSWORD="${BITBUCKET_APP_PASSWORD:-$(cat /usr/local/rhtap-cli-install/bitbucket-app-password)}"

    "${TSSC_BINARY}" integration --kube-config "$KUBECONFIG" bitbucket --host="${BITBUCKET_HOST}" --username="${BITBUCKET_USERNAME}" --app-password="${BITBUCKET_APP_PASSWORD}" --force
  fi
}

# Workaround: This function has to be called before tssc import "installer/config.yaml" into cluster.
# Currently, the tssc `config` subcommand lacks the ability to modify property values stored in cluster
disable_tpa() {
  # if "remote" is in tpa_config array, then disable TPA installation
  # Update the enabled flag from true to false (line 7 in config.yaml)
  if [[ " ${tpa_config[*]} " =~ " remote " ]]; then
    echo "[INFO] Disable TPA installation in TSSC configuration"
    yq -i '.tssc.products[] |= select(.name == "Trusted Profile Analyzer").enabled = false' "${config_file}"
  else
    echo "[INFO] TPA is set to local, keeping enabled flag as true"
  fi
}

tpa_integration() {
  if [[ " ${tpa_config[*]} " =~ " remote " ]]; then
    echo "[INFO] Configure a remote TPA integration into TSSC"

    BOMBASTIC_API_URL="${BOMBASTIC_API_URL:-$(cat /usr/local/rhtap-cli-install/bombastic-api-url)}"
    OIDC_CLIENT_ID="${OIDC_CLIENT_ID:-$(cat /usr/local/rhtap-cli-install/oidc-client-id)}"
    OIDC_CLIENT_SECRET="${OIDC_CLIENT_SECRET:-$(cat /usr/local/rhtap-cli-install/oidc-client-secret)}"
    OIDC_ISSUER_URL="${OIDC_ISSUER_URL:-$(cat /usr/local/rhtap-cli-install/oidc-issuer-url)}"

    "${TSSC_BINARY}" integration --kube-config "$KUBECONFIG" trustification --bombastic-api-url="${BOMBASTIC_API_URL}" --oidc-client-id="${OIDC_CLIENT_ID}" --oidc-client-secret="${OIDC_CLIENT_SECRET}" --oidc-issuer-url="${OIDC_ISSUER_URL}" --supported-cyclonedx-version="${SUPPORTED_CYCLONEDX_VERSION}" --force
  fi
}

# Workaround: This function has to be called before tssc import "installer/config.yaml" into cluster.
# Currently, the tssc `config` subcommand lacks the ability to modify property values stored in cluster
disable_tas() {
  # if "remote" is in tas_config array, then disable TAS installation
  # Update the enabled flag from true to false
  if [[ " ${tas_config[*]} " =~ " remote " ]]; then
    echo "[INFO] Disable TAS installation in TSSC configuration"
    yq -i '.tssc.products[] |= select(.name == "Trusted Artifact Signer").enabled = false' "${config_file}"
  else
    echo "[INFO] TAS is set to local, keeping enabled flag as true"
  fi
}

tas_integration() {
  if [[ " ${tas_config[*]} " =~ " remote " ]]; then
    echo "[INFO] Configure a remote TAS integration into TSSC"

    TAS__REKOR__URL="${TAS__REKOR__URL:-$(cat /usr/local/rhtap-cli-install/tas-rekor-url)}"
    TAS__TUF__URL="${TAS__TUF__URL:-$(cat /usr/local/rhtap-cli-install/tas-tuf-url)}"

    "${TSSC_BINARY}" integration --kube-config "$KUBECONFIG" trusted-artifact-signer --rekor-url="${TAS__REKOR__URL}" --tuf-url="${TAS__TUF__URL}" --force
  fi
}

artifactory_integration() {
  if [[ " ${registry_config[*]} " =~ " artifactory " ]]; then
    echo "[INFO] Configure Artifactory integration into TSSC"

    ARTIFACTORY_URL="${ARTIFACTORY_URL:-$(cat /usr/local/rhtap-cli-install/artifactory-url)}"
    ARTIFACTORY_TOKEN="${ARTIFACTORY_TOKEN:-$(cat /usr/local/rhtap-cli-install/artifactory-token)}"
    ARTIFACTORY_DOCKERCONFIGJSON="${ARTIFACTORY_DOCKERCONFIGJSON:-$(cat /usr/local/rhtap-cli-install/artifactory-dockerconfig-json)}"
    "${TSSC_BINARY}" integration --kube-config "$KUBECONFIG" artifactory --url="${ARTIFACTORY_URL}" --token="${ARTIFACTORY_TOKEN}" --dockerconfigjson="${ARTIFACTORY_DOCKERCONFIGJSON}" --force
  fi
}

nexus_integration() {
  if [[ " ${registry_config[*]} " =~ " nexus " ]]; then
    echo "[INFO] Configure Nexus integration into TSSC"

    NEXUS_URL="${NEXUS_URL:-$(cat /usr/local/rhtap-cli-install/nexus-ui-url)}"
    NEXUS_DOCKERCONFIGJSON="${NEXUS_DOCKERCONFIGJSON:-$(cat /usr/local/rhtap-cli-install/nexus-dockerconfig-json)}"
    "${TSSC_BINARY}" integration --kube-config "$KUBECONFIG" nexus --url="${NEXUS_URL}" --dockerconfigjson="${NEXUS_DOCKERCONFIGJSON}" --force
  fi
}

run_pre_release() {
  if [[ -n "${PRE_RELEASE:-}" ]]; then
    echo "[INFO] Running pre-release configuration for product: $PRE_RELEASE"
    
    # Get the script directory to find pre-release.sh
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    PRE_RELEASE_SCRIPT="$SCRIPT_DIR/../../hack/pre-release/pre-release.sh"
    
    # Check if pre-release.sh exists
    if [[ ! -f "$PRE_RELEASE_SCRIPT" ]]; then
      echo "[ERROR] pre-release.sh not found at $PRE_RELEASE_SCRIPT"
      exit 1
    fi
    
    # Map product names to what pre-release.sh expects
    case "${PRE_RELEASE^^}" in
      RHDH)
        PRODUCT_ARG="rhdh"
        PRE_RELEASE_CMD=("$PRE_RELEASE_SCRIPT" --product "$PRODUCT_ARG")
        ;;
      TPA)
        PRODUCT_ARG="rhtpa"
        PRE_RELEASE_CMD=("$PRE_RELEASE_SCRIPT" --product "$PRODUCT_ARG")
        # Add GitHub token if available (needed for private repos)
        # Use GITHUB_TOKEN if set, otherwise fall back to GITOPS_GIT_TOKEN
        GITHUB_TOKEN_VALUE="${GITHUB_TOKEN:-${GITOPS_GIT_TOKEN:-}}"
        if [[ -n "$GITHUB_TOKEN_VALUE" ]]; then
          # Export as environment variable for pre-release.sh to use
          export GITHUB_TOKEN="$GITHUB_TOKEN_VALUE"
          PRE_RELEASE_CMD+=("--github-token" "$GITHUB_TOKEN_VALUE")
        fi
        ;;
      TAS)
        PRODUCT_ARG="rhtas"
        # TAS requires --tas-release-path only if neither GITHUB_TOKEN nor TAS_VERSION is provided
        # Use GITHUB_TOKEN if set, otherwise fall back to GITOPS_GIT_TOKEN
        GITHUB_TOKEN_VALUE="${GITHUB_TOKEN:-${GITOPS_GIT_TOKEN:-}}"
        if [[ -z "${TAS_RELEASE_PATH:-}" ]] && [[ -z "${GITHUB_TOKEN_VALUE:-}" ]] && [[ -z "${TAS_VERSION:-}" ]]; then
          echo "[ERROR] TAS_RELEASE_PATH environment variable is required when PRE_RELEASE=TAS"
          echo "[ERROR] unless GITHUB_TOKEN (or GITOPS_GIT_TOKEN) or TAS_VERSION is provided"
          echo "[ERROR] Example: export TAS_RELEASE_PATH='https://github.com/securesign/releases/blob/release-1.3.1/1.3.1/stable'"
          echo "[ERROR] Or: export GITHUB_TOKEN='ghp_xxxxx' (for auto-detection)"
          echo "[ERROR] Or: export TAS_VERSION='1.3.1' (with GITHUB_TOKEN for private repos)"
          exit 1
        fi
        PRE_RELEASE_CMD=("$PRE_RELEASE_SCRIPT" --product "$PRODUCT_ARG")
        # Add --tas-release-path if provided
        if [[ -n "${TAS_RELEASE_PATH:-}" ]]; then
          PRE_RELEASE_CMD+=("--tas-release-path" "$TAS_RELEASE_PATH")
        fi
        # Add --tas-release-version if provided
        if [[ -n "${TAS_VERSION:-}" ]]; then
          PRE_RELEASE_CMD+=("--tas-release-version" "$TAS_VERSION")
        fi
        # Add --tas-operator-version if provided
        if [[ -n "${TAS_OPERATOR_VERSION:-}" ]]; then
          PRE_RELEASE_CMD+=("--tas-operator-version" "$TAS_OPERATOR_VERSION")
        fi
        # Add GitHub token if available (needed for private repos and auto-detection)
        if [[ -n "$GITHUB_TOKEN_VALUE" ]]; then
          # Export as environment variable for pre-release.sh to use
          export GITHUB_TOKEN="$GITHUB_TOKEN_VALUE"
          PRE_RELEASE_CMD+=("--github-token" "$GITHUB_TOKEN_VALUE")
        fi
        ;;
      *)
        echo "[ERROR] Unknown product for pre-release: $PRE_RELEASE. Supported values: RHDH, TPA, TAS"
        exit 1
        ;;
    esac
    
    echo "[INFO] Executing pre-release.sh with product: $PRODUCT_ARG"
    if [[ "${PRE_RELEASE^^}" == "TAS" ]]; then
      if [[ -n "${TAS_RELEASE_PATH:-}" ]]; then
        echo "[INFO] Using TAS release path: $TAS_RELEASE_PATH"
      else
        echo "[INFO] TAS release path will be auto-detected"
      fi
    fi
    bash "${PRE_RELEASE_CMD[@]}"

  else
    echo "[INFO] pre-release parameter not set, skipping pre-release configuration"
  fi
}

create_cluster_config() {
  echo "[INFO] Creating the installer's cluster configuration"
  update_dh_catalog_url
  update_dh_auth_config
  update_dh_namespace_prefixes
  disable_acs
  disable_tpa
  disable_tas
  
  set -x
  cat "$config_file"
  set +x

  echo "[INFO] Applying the cluster configuration, and showing the 'config.yaml'"
  set -x
    "${TSSC_BINARY}" config --kube-config "$KUBECONFIG" --get --create "$config_file" --force
  set +x
  
  echo "[INFO] Cluster configuration created successfully"
  
  # Ensure the installer namespace exists for integrations
  echo "[INFO] Ensuring installer namespace exists for integrations"
  INSTALLER_NAMESPACE=$(yq '.tssc.namespace' "${config_file}" 2>/dev/null || echo "tssc")
  kubectl get namespace "${INSTALLER_NAMESPACE}" >/dev/null 2>&1 || kubectl create namespace "${INSTALLER_NAMESPACE}"
}

configure_integrations() {
  echo "[INFO] Configuring required integrations before deployment"
  
  # Configure integrations in the order that ensures dependencies are met
  # SCM integrations first (required by tssc-app-namespaces)
  github_integration
  gitlab_integration
  bitbucket_integration
  
  # Registry integrations
  quay_integration
  artifactory_integration
  nexus_integration
  
  # Pipeline integrations
  jenkins_integration
  azure_integration
  
  # Security integrations (required by tssc-app-namespaces when remote)
  # These functions check internally if acs_config, tpa_config, or tas_config is "remote"
  acs_integration
  tpa_integration
  tas_integration
  
  echo "[INFO] Verifying required integrations are configured"
  echo "[INFO] Integration secrets in 'tssc' namespace:"
  kubectl -n tssc get secret 2>/dev/null | grep -E "(github|gitlab|bitbucket|acs|trustification)" || echo "  (some secrets may not exist yet)"
}

install_tssc() {
  echo "[INFO] Start installing TSSC"
  if [[ -n "${PRE_RELEASE:-}" ]]; then
    make build
    TSSC_BINARY="$(pwd)/bin/tssc"
  fi
  echo "[INFO] Using tssc binary: ${TSSC_BINARY}"

  echo "[INFO] Print out the content of 'values.yaml.tpl'"
  set -x
  cat "$tpl_file"
  set +x

  echo "[INFO] Running 'tssc deploy' command..."
  set -x
    "${TSSC_BINARY}" deploy --timeout 35m --values-template "$tpl_file" --kube-config "$KUBECONFIG"
  set +x

  homepage_url=https://$(kubectl -n tssc-dh get route backstage-developer-hub -o  'jsonpath={.spec.host}')

  echo "[INFO] homepage_url=$homepage_url"

  echo "[INFO] Print out the integration secrets in 'tssc' namespace"
  kubectl -n tssc get secret

  if [[ -n "${PRE_RELEASE:-}" ]]; then
    echo "[INFO] Installed operators and versions (pre-release: $PRE_RELEASE):"
    kubectl get csv -A -o jsonpath='{range .items[*]}{.metadata.namespace}{" "}{.metadata.name}{"\n"}{end}' 2>/dev/null \
      | awk '$1 ~ /^tssc/ {print $2}' | sort -u || true
  fi
}

ci_enabled
run_pre_release
create_cluster_config
configure_integrations
install_tssc
