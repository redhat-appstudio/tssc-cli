#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Source common functions
SCRIPT_DIR="$(
    cd "$(dirname "$0")" >/dev/null || exit
    pwd
)"
# shellcheck source=hack/pre-release/pre-release-common.sh
source "$SCRIPT_DIR/pre-release-common.sh"

usage() {
    echo "
Usage:
    ${0##*/} [options]

Optional arguments:
    -t, --github-token TOKEN
        GitHub personal access token for accessing private repositories.
        Can also be set via GITHUB_TOKEN environment variable.
    -d, --debug
        Activate tracing/debug mode.
    -h, --help
        Display this message.

Examples:
    ${0##*/} --github-token ghp_xxxxx
" >&2
}

parse_args() {
    init_github_token
    while [[ $# -gt 0 ]]; do
        case $1 in
        -t|--github-token)
            if [[ -z "${2:-}" ]]; then
                echo "[ERROR] GitHub token needs to be specified after '--github-token'." >&2
                usage
                exit 1
            fi
            GITHUB_TOKEN="$2"
            export GITHUB_TOKEN
            shift
            ;;
        -d|--debug)
            set -x
            DEBUG="--debug"
            export DEBUG
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "[ERROR] Unknown argument: $1" >&2
            usage
            exit 1
            ;;
        esac
        shift
    done
}

# TODO: Investigate how to get latest released image tag for TPA
# The catalog image format is: quay.io/redhat-user-workloads/trusted-content-tenant/rhtpa-fbc-v{ocp_version}:{digest}
# Potential approaches:
# 1. Query Quay.io API for the latest tag in the rhtpa-fbc-v{ocp_version} repository
# 2. Query GitHub releases if the digest is stored in a release
# 3. Use a fixed digest if it's stable across releases
# For now, we use the fixed digest: 5d1f0a545dc1a85dad8349c0aa5369384a5aa9d0
get_latest_tpa_image_digest() {
    # shellcheck disable=SC2034
    # ocp_version parameter is reserved for future use when implementing digest lookup
    local ocp_version=$1
    # TODO: Implement logic to fetch latest digest from Quay.io or GitHub
    # For now, return the known digest
    #echo "5d1f0a545dc1a85dad8349c0aa5369384a5aa9d0"
    echo "v1.1.1-rc"
}

configure_tpa() {
    echo "[INFO] Configuring TPA (Trusted Profile Analyzer) Operator for pre-release testing" >&2
    
    # Step 1: Get cluster version for catalogSource image
    CLUSTER_VERSION=$(get_cluster_version)
    if [[ -z "$CLUSTER_VERSION" ]]; then
        echo "[ERROR] Failed to determine cluster version" >&2
        exit 1
    fi
    echo "[INFO] Detected OpenShift cluster version: $CLUSTER_VERSION" >&2
    
    # Convert version format: 4.20 -> 420 (remove dot for image tag)
    OCP_VERSION_FOR_IMAGE=$(echo "$CLUSTER_VERSION" | tr -d '.')
    echo "[INFO] Using OCP version for catalog image: $OCP_VERSION_FOR_IMAGE" >&2
    
    # Step 2: Create ImageDigestMirrorSet
    echo "[INFO] Step 2: Creating ImageDigestMirrorSet..." >&2
    IMAGE_MIRROR_FILE="$(mktemp)"
    cat > "$IMAGE_MIRROR_FILE" <<EOF
apiVersion: config.openshift.io/v1
kind: ImageDigestMirrorSet
metadata:
  name: rhtap-tp
spec:
  imageDigestMirrors:
    - mirrorSourcePolicy: NeverContactSource
      mirrors:
        - quay.io/redhat-user-workloads/trusted-content-tenant/rhtpa-product-0-4-z
      source: registry.redhat.io/rhtpa/rhtpa-trustification-service-rhel9
    - mirrorSourcePolicy: NeverContactSource
      mirrors:
        - quay.io/redhat-user-workloads/trusted-content-tenant/operator-1-1-z
      source: registry.redhat.io/rhtpa/rhtpa-rhel9-operator
    - mirrorSourcePolicy: NeverContactSource
      mirrors:
        - quay.io/redhat-user-workloads/trusted-content-tenant/operator-bundle-1-1-z
      source: registry.redhat.io/rhtpa/rhtpa-operator-bundle-rhel9
    - mirrorSourcePolicy: NeverContactSource
      mirrors:
        - quay.io/redhat-user-workloads/trusted-content-tenant/operator-bundle-1-1-z
      source: registry.redhat.io/rhtpa/rhtpa-operator-bundle
EOF
    
    echo "[INFO] Applying ImageDigestMirrorSet..." >&2
    if ! oc apply -f "$IMAGE_MIRROR_FILE"; then
        echo "[ERROR] Failed to apply ImageDigestMirrorSet" >&2
        rm -f "$IMAGE_MIRROR_FILE"
        exit 1
    fi
    rm -f "$IMAGE_MIRROR_FILE"
    echo "[INFO] ✓ ImageDigestMirrorSet applied successfully" >&2
    
    # Step 3: Create CatalogSource with FBC image
    echo "[INFO] Step 3: Creating CatalogSource..." >&2
    # CatalogSource should be in openshift-marketplace to match values.yaml expectations
    CATALOG_NAMESPACE="${CATALOG_NAMESPACE:-openshift-marketplace}"
    # TPA operator can only be installed in a single namespace (not AllNamespaces)
    # Use tssc-tpa namespace as specified in values.yaml
    NAMESPACE="${NAMESPACE:-tssc-tpa}"
    
    # Ensure the namespace exists
    echo "[INFO] Ensuring namespace $NAMESPACE exists..." >&2
    if ! oc get namespace "$NAMESPACE" >/dev/null 2>&1; then
        echo "[INFO] Creating namespace $NAMESPACE..." >&2
        if ! oc create namespace "$NAMESPACE"; then
            echo "[ERROR] Failed to create namespace $NAMESPACE" >&2
            exit 1
        fi
    fi
    
    # Get latest image digest (TODO: implement auto-detection)
    IMAGE_DIGEST=$(get_latest_tpa_image_digest "$OCP_VERSION_FOR_IMAGE")
    CATALOG_IMAGE="quay.io/redhat-user-workloads/trusted-content-tenant/rhtpa-fbc-v${OCP_VERSION_FOR_IMAGE}:${IMAGE_DIGEST}"
    
    CATALOG_FILE="$(mktemp)"
    cat > "$CATALOG_FILE" <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: rhtpa-fbc-pre
  namespace: ${CATALOG_NAMESPACE}
spec:
  displayName: rhtpa-fbc-pre
  image: ${CATALOG_IMAGE}
  publisher: rhtpa-eng-konflux
  sourceType: grpc
EOF
    
    echo "[INFO] Using catalog image: $CATALOG_IMAGE" >&2
    echo "[INFO] Applying CatalogSource..." >&2
    if ! oc apply -f "$CATALOG_FILE"; then
        echo "[ERROR] Failed to apply CatalogSource" >&2
        rm -f "$CATALOG_FILE"
        exit 1
    fi
    rm -f "$CATALOG_FILE"
    echo "[INFO] ✓ CatalogSource applied successfully" >&2
    
    # Step 4: Create OperatorGroup for SingleNamespace mode
    echo "[INFO] Step 4: Creating OperatorGroup for SingleNamespace installation..." >&2
    OPERATORGROUP_FILE="$(mktemp)"
    cat > "$OPERATORGROUP_FILE" <<EOF
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: rhtpa-operator-group
  namespace: ${NAMESPACE}
spec:
  targetNamespaces:
    - ${NAMESPACE}
EOF
    
    echo "[INFO] Applying OperatorGroup..." >&2
    if ! oc apply -f "$OPERATORGROUP_FILE"; then
        echo "[ERROR] Failed to apply OperatorGroup" >&2
        rm -f "$OPERATORGROUP_FILE"
        exit 1
    fi
    rm -f "$OPERATORGROUP_FILE"
    echo "[INFO] ✓ OperatorGroup applied successfully" >&2
    
    # Step 5: Create Subscription
    echo "[INFO] Step 5: Creating Subscription..." >&2
    SUBSCRIPTION_FILE="$(mktemp)"
    cat > "$SUBSCRIPTION_FILE" <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: rhtpa-operator
  namespace: ${NAMESPACE}
spec:
  channel: stable-v1.1
  name: rhtpa-operator
  source: rhtpa-fbc-pre
  sourceNamespace: ${CATALOG_NAMESPACE}
  installPlanApproval: Automatic
  startingCSV: rhtpa-operator.v1.1.0
EOF
    
    echo "[INFO] Applying Subscription with channel: stable-v1.1, version: 1.1.0" >&2
    if ! oc apply -f "$SUBSCRIPTION_FILE"; then
        echo "[ERROR] Failed to apply Subscription" >&2
        rm -f "$SUBSCRIPTION_FILE"
        exit 1
    fi
    rm -f "$SUBSCRIPTION_FILE"
    echo "[INFO] ✓ Subscription applied successfully" >&2
    
    echo "[INFO] Operator Lifecycle Manager (OLM) is initiating TPA Operator deployment..." >&2
    
    # Extract subscription information for configure_subscription
    CHANNEL="stable-v1.1"
    SOURCE="rhtpa-fbc-pre"
    SUBSCRIPTION="trustedProfileAnalyzer"
    export SUBSCRIPTION
    export CHANNEL
    export SOURCE
    
    # Update config.yaml to disable subscription management for TPA
    config_file="$PROJECT_DIR/installer/config.yaml"
    if [[ -f "$config_file" ]]; then
        echo "[INFO] Updating config.yaml to disable TPA subscription management" >&2
        yq -i '.tssc.products[] |= (select(.name == "Trusted Profile Analyzer") | .properties.manageSubscription = false)' "$config_file"
        echo "[INFO] ✓ TPA subscription management disabled in config.yaml" >&2
    else
        echo "[WARNING] config.yaml not found at $config_file, skipping subscription management update" >&2
    fi
}

# configure_subscription is now in pre-release-common.sh

main() {
    parse_args "$@"
    
    configure_tpa
    # Only configure subscription if SUBSCRIPTION variable is set
    if [[ -n "${SUBSCRIPTION:-}" ]]; then
        configure_subscription
    fi
    echo "Done" >&2
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
fi
