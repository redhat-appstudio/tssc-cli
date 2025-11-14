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

# Export after setting
export acs_config tpa_config registry_config scm_config pipeline_config

echo "[INFO] acs_config=(${acs_config[*]})"
echo "[INFO] tpa_config=(${tpa_config[*]})"
echo "[INFO] registry_config=(${registry_config[*]})"
echo "[INFO] scm_config=(${scm_config[*]})"
echo "[INFO] pipeline_config=(${pipeline_config[*]})"

tpl_file="installer/charts/values.yaml.tpl"
config_file="installer/config.yaml"

ci_enabled() {
  echo "[INFO] Turn ci to true, this is required when you perform rhtap-e2e automation test against TSSC"
  yq -i '.tssc.settings.ci.debug = true' "${config_file}"
}

update_dh_catalog_url() {
  # if DEVELOPER_HUB__CATALOG__URL is not empty string, then update the catalog url
  if [[ -n "${DEVELOPER_HUB__CATALOG__URL}" ]]; then
    echo "[INFO] Update dh catalog url with $DEVELOPER_HUB__CATALOG__URL"
    yq -i '.tssc.products[] |= select(.name == "Developer Hub").properties.catalogURL=strenv(DEVELOPER_HUB__CATALOG__URL)' "${config_file}"
  fi
}

# Workaround: This function has to be called before tssc import "installer/config.yaml" into cluster.
# Currently, the tssc `config` subcommand lacks the ability to modify property values stored in config.yaml.
github_integration() {
  # Check if "github" is in scm_config array
  if [[ " ${scm_config[*]} " =~ " github " ]]; then
    echo "[INFO] Config Github integration with TSSC"

    GITHUB__APP__ID="${GITHUB__APP__ID:-$(cat /usr/local/rhtap-cli-install/rhdh-github-app-id)}"
    GITHUB__APP__CLIENT__ID="${GITHUB__APP__CLIENT__ID:-$(cat /usr/local/rhtap-cli-install/rhdh-github-client-id)}"
    GITHUB__APP__CLIENT__SECRET="${GITHUB__APP__CLIENT__SECRET:-$(cat /usr/local/rhtap-cli-install/rhdh-github-client-secret)}"
    GITHUB__APP__PRIVATE_KEY="${GITHUB__APP__PRIVATE_KEY:-$(base64 -d < /usr/local/rhtap-cli-install/rhdh-github-private-key | sed 's/^/        /')}"
    GITOPS__GIT_TOKEN="${GITOPS__GIT_TOKEN:-$(cat /usr/local/rhtap-cli-install/github_token)}"
    GITHUB__APP__WEBHOOK__SECRET="${GITHUB__APP__WEBHOOK__SECRET:-$(cat /usr/local/rhtap-cli-install/rhdh-github-webhook-secret)}"

    cat << EOF | kubectl apply -f -
kind: Secret
type: Opaque
apiVersion: v1
metadata:
    name: tssc-github-integration
    namespace: tssc
stringData:
    id: "$GITHUB__APP__ID"
    clientId: "$GITHUB__APP__CLIENT__ID"
    clientSecret: "$GITHUB__APP__CLIENT__SECRET"
    host: github.com
    pem: |
$(printf "%s\n" "${GITHUB__APP__PRIVATE_KEY}" | sed 's/^/        /')
    token: "$GITOPS__GIT_TOKEN"
    webhookSecret: "$GITHUB__APP__WEBHOOK__SECRET"
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

gitlab_integration() {
  if [[ " ${scm_config[*]} " =~ " gitlab " ]]; then
    echo "[INFO] Configure Gitlab integration into TSSC"

    GITLAB__TOKEN="${GITLAB__TOKEN:-$(cat /usr/local/rhtap-cli-install/gitlab_token)}"

    GITLAB__APP__ID="${GITLAB__APP__ID:-$(cat /usr/local/rhtap-cli-install/gitlab-app-id)}"
    GITLAB__APP_SECRET="${GITLAB__APP_SECRET:-$(cat /usr/local/rhtap-cli-install/gitlab-app-secret)}"
    GITLAB__GROUP="${GITLAB__GROUP:-$(cat /usr/local/rhtap-cli-install/gitlab-group)}"

    "${TSSC_BINARY}" integration --kube-config "$KUBECONFIG" gitlab --token="${GITLAB__TOKEN}" --app-id="${GITLAB__APP__ID}" --app-secret="${GITLAB__APP_SECRET}" --group="${GITLAB__GROUP}"
  fi
}

quay_integration() {
  if [[ " ${registry_config[*]} " =~  quay ]]; then
    echo "[INFO] Configure quay integration into TSSC"

    QUAY__DOCKERCONFIGJSON="${QUAY__DOCKERCONFIGJSON:-$(cat /usr/local/rhtap-cli-install/quay-dockerconfig-json)}"
    QUAY__API_TOKEN="${QUAY__API_TOKEN:-$(cat /usr/local/rhtap-cli-install/quay-api-token)}"

    "${TSSC_BINARY}" integration --kube-config "$KUBECONFIG" quay --url="https://quay.io" --dockerconfigjson="${QUAY__DOCKERCONFIGJSON}" --token="${QUAY__API_TOKEN}"
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

    ACS__CENTRAL_ENDPOINT="${ACS__CENTRAL_ENDPOINT:-$(cat /usr/local/rhtap-cli-install/acs-central-endpoint)}"
    ACS__API_TOKEN="${ACS__API_TOKEN:-$(cat /usr/local/rhtap-cli-install/acs-api-token)}"

    "${TSSC_BINARY}" integration --kube-config "$KUBECONFIG" acs --endpoint="${ACS__CENTRAL_ENDPOINT}" --token="${ACS__API_TOKEN}"
  fi
}

bitbucket_integration() {
  if [[ " ${scm_config[*]} " =~ " bitbucket " ]]; then
    echo "[INFO] Configure Bitbucket integration into TSSC"

    BITBUCKET_USERNAME="${BITBUCKET_USERNAME:-$(cat /usr/local/rhtap-cli-install/bitbucket-username)}"
    BITBUCKET_APP_PASSWORD="${BITBUCKET_APP_PASSWORD:-$(cat /usr/local/rhtap-cli-install/bitbucket-app-password)}"

    "${TSSC_BINARY}" integration --kube-config "$KUBECONFIG" bitbucket --host="${BITBUCKET_HOST}" --username="${BITBUCKET_USERNAME}" --app-password="${BITBUCKET_APP_PASSWORD}"
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

    "${TSSC_BINARY}" integration --kube-config "$KUBECONFIG" trustification --bombastic-api-url="${BOMBASTIC_API_URL}" --oidc-client-id="${OIDC_CLIENT_ID}" --oidc-client-secret="${OIDC_CLIENT_SECRET}" --oidc-issuer-url="${OIDC_ISSUER_URL}" --supported-cyclonedx-version="${SUPPORTED_CYCLONEDX_VERSION}"
  fi
}

artifactory_integration() {
  if [[ " ${registry_config[*]} " =~ " artifactory " ]]; then
    echo "[INFO] Configure Artifactory integration into TSSC"

    ARTIFACTORY_URL="${ARTIFACTORY_URL:-$(cat /usr/local/rhtap-cli-install/artifactory-url)}"
    ARTIFACTORY_TOKEN="${ARTIFACTORY_TOKEN:-$(cat /usr/local/rhtap-cli-install/artifactory-token)}"
    ARTIFACTORY_DOCKERCONFIGJSON="${ARTIFACTORY_DOCKERCONFIGJSON:-$(cat /usr/local/rhtap-cli-install/artifactory-dockerconfig-json)}"
    "${TSSC_BINARY}" integration --kube-config "$KUBECONFIG" artifactory --url="${ARTIFACTORY_URL}" --token="${ARTIFACTORY_TOKEN}" --dockerconfigjson="${ARTIFACTORY_DOCKERCONFIGJSON}"
  fi
}

nexus_integration() {
  if [[ " ${registry_config[*]} " =~ " nexus " ]]; then
    echo "[INFO] Configure Nexus integration into TSSC"

    NEXUS_URL="${NEXUS_URL:-$(cat /usr/local/rhtap-cli-install/nexus-ui-url)}"
    NEXUS_DOCKERCONFIGJSON="${NEXUS_DOCKERCONFIGJSON:-$(cat /usr/local/rhtap-cli-install/nexus-dockerconfig-json)}"
    "${TSSC_BINARY}" integration --kube-config "$KUBECONFIG" nexus --url="${NEXUS_URL}" --dockerconfigjson="${NEXUS_DOCKERCONFIGJSON}"
  fi
}

create_cluster_config() {
  echo "[INFO] Creating the installer's cluster configuration"
  update_dh_catalog_url
  disable_acs
  disable_tpa
  
  set -x
  cat "$config_file"
  set +x

  echo "[INFO] Applying the cluster configuration, and showing the 'config.yaml'"
  set -x
    "${TSSC_BINARY}" config --kube-config "$KUBECONFIG" --get --create "$config_file"
  set +x
  
  echo "[INFO] Cluster configuration created successfully"
}


install_tssc() {
  echo "[INFO] Start installing TSSC"

  echo "[INFO] Print out the content of 'values.yaml.tpl'"
  set -x
  cat "$tpl_file"
  set +x

  jenkins_integration
  tpa_integration
  acs_integration
  github_integration
  gitlab_integration
  bitbucket_integration
  quay_integration
  artifactory_integration
  nexus_integration

  echo "[INFO] Print out the integration secrets in 'tssc' namespace"
  kubectl -n tssc get secret 
  kubectl -n tssc get secret tssc-github-integration -o yaml

  echo "[INFO] Running 'tssc deploy' command..."
  set -x
    "${TSSC_BINARY}" deploy --timeout 35m --values-template "$tpl_file" --kube-config "$KUBECONFIG"
  set +x

  homepage_url=https://$(kubectl -n tssc-dh get route backstage-developer-hub -o  'jsonpath={.spec.host}')

  echo "[INFO] homepage_url=$homepage_url"

  echo "[INFO] Print out the integration secrets in 'tssc' namespace"
  kubectl -n tssc get secret 
}

ci_enabled
create_cluster_config
install_tssc
