#!/usr/bin/env bash
# Common functions and variables for pre-release scripts

# Initialize script and project directories
SCRIPT_DIR="$(
    cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null || exit
    pwd
)"

PROJECT_DIR="$(
    cd "$SCRIPT_DIR/../.." >/dev/null || exit
    pwd
)"

# Get OpenShift cluster version (e.g., 4.20, 4.19)
get_cluster_version() {
    oc get clusterversion version -o jsonpath='{.status.desired.version}' | cut -d. -f1,2
}

# Path to values.yaml that owns .subscriptions.<key> (bundle-local subscription charts).
subscription_values_file_for() {
    case "$1" in
    openshiftKeycloak) echo "$PROJECT_DIR/installer/bundles/iam/charts/tssc-iam-subscriptions/values.yaml" ;;
    openshiftGitOps) echo "$PROJECT_DIR/installer/bundles/gitops/charts/tssc-gitops-subscriptions/values.yaml" ;;
    openshiftPipelines) echo "$PROJECT_DIR/installer/bundles/pipelines/charts/tssc-pipelines-subscriptions/values.yaml" ;;
    openshiftTrustedArtifactSigner) echo "$PROJECT_DIR/installer/bundles/tas/charts/tssc-tas-subscriptions/values.yaml" ;;
    advancedClusterSecurity) echo "$PROJECT_DIR/installer/bundles/acs/charts/tssc-acs-subscriptions/values.yaml" ;;
    developerHub) echo "$PROJECT_DIR/installer/bundles/dh/charts/tssc-dh-subscriptions/values.yaml" ;;
    trustedProfileAnalyzer) echo "$PROJECT_DIR/installer/bundles/tpa/charts/tssc-tpa-subscriptions/values.yaml" ;;
    *)
        echo "[ERROR] Unknown subscription key: $1" >&2
        return 1
        ;;
    esac
}

# Configure subscription in values.yaml
# This function should be called after SUBSCRIPTION, CHANNEL, and SOURCE are set
configure_subscription() {
    # Prepare for pre-release install capabilities
    local subscription_values_file
    subscription_values_file="$(subscription_values_file_for "${SUBSCRIPTION:-}")"
    
    # For TAS, set source to rhtas-operator (without 's' at the end)
    if [[ "${SUBSCRIPTION:-}" == "openshiftTrustedArtifactSigner" ]]; then
        SOURCE="rhtas-operator"
    fi

    yq -i "
        .subscriptions.$SUBSCRIPTION.channel = \"$CHANNEL\",
        .subscriptions.$SUBSCRIPTION.source = \"$SOURCE\"
    " "$subscription_values_file"

    cat "$subscription_values_file"
}

# Initialize GITHUB_TOKEN from environment or GITOPS_GIT_TOKEN
init_github_token() {
    GITHUB_TOKEN="${GITHUB_TOKEN:-${GITOPS_GIT_TOKEN:-}}"
}
