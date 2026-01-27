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

# Configure subscription in values.yaml
# This function should be called after SUBSCRIPTION, CHANNEL, and SOURCE are set
configure_subscription() {
    # Prepare for pre-release install capabilities
    local subscription_values_file="$PROJECT_DIR/installer/charts/tssc-subscriptions/values.yaml"
    
    # For TAS, set source to rhtas-operator (without 's' at the end)
    if [[ "${SUBSCRIPTION:-}" == "openshiftTrustedArtifactSigner" ]]; then
        SOURCE="rhtas-operator"
    fi

    yq -i "
        .subscriptions.$SUBSCRIPTION.channel = \"$CHANNEL\",
        .subscriptions.$SUBSCRIPTION.source = \"$SOURCE\"
    " "$subscription_values_file"

    cat $subscription_values_file
}

# Initialize GITHUB_TOKEN from environment or GITOPS_GIT_TOKEN
init_github_token() {
    GITHUB_TOKEN="${GITHUB_TOKEN:-${GITOPS_GIT_TOKEN:-}}"
}
