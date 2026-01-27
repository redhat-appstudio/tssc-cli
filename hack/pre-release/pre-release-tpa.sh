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
    -d, --debug
        Activate tracing/debug mode.
    -h, --help
        Display this message.
" >&2
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
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
# The catalog image format is: quay.io/redhat-user-workloads/trusted-content-tenant/rhtpa-fbc-v{ocp_version}:{version_tag}
# Example: quay.io/redhat-user-workloads/trusted-content-tenant/rhtpa-fbc-v417:v1.1.1-rc
# Potential approaches:
# 1. Query Quay.io API for the latest tag in the rhtpa-fbc-v{ocp_version} repository
# 2. Query GitHub releases if the version tag is stored in a release
# 3. Use a fixed version tag if it's stable across releases
# For now, we use the fixed version tag: v1.1.1-rc
get_latest_tpa_image_tag() {
    # shellcheck disable=SC2034
    # ocp_version parameter is reserved for future use when implementing tag lookup
    local ocp_version=$1
    # TODO: Implement logic to fetch latest tag from Quay.io or GitHub
    # For now, return the known version tag for RHTPA 2.2.1 pre-release
    echo "v1.1.2-rc"
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
    
    # Get latest image tag (TODO: implement auto-detection)
    IMAGE_TAG=$(get_latest_tpa_image_tag "$OCP_VERSION_FOR_IMAGE")
    CATALOG_IMAGE="quay.io/redhat-user-workloads/trusted-content-tenant/rhtpa-fbc-v${OCP_VERSION_FOR_IMAGE}:${IMAGE_TAG}"
    
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
    
    # Step 4: Update values.yaml to use pre-release catalog source and channel
    # SingleNamespace mode is configured in values.yaml by setting namespace to openshift-operators
    # with targetNamespaces: [tssc-tpa] - no manual OperatorGroup/Subscription creation needed
    echo "[INFO] Step 4: Updating tssc-cli configuration for pre-release TPA..." >&2
    
    values_file="$PROJECT_DIR/installer/charts/tssc-subscriptions/values.yaml"
    if [[ -f "$values_file" ]]; then
        echo "[INFO] Updating values.yaml to use pre-release TPA catalog source and channel" >&2
        echo "[INFO] Setting source to: rhtpa-fbc-pre, channel to: stable-v1.1" >&2
        yq -i ".subscriptions.trustedProfileAnalyzer.source = \"rhtpa-fbc-pre\"" "$values_file"
        yq -i ".subscriptions.trustedProfileAnalyzer.channel = \"stable-v1.1\"" "$values_file"
        echo "[INFO] ✓ TPA subscription source and channel updated in values.yaml" >&2
        echo "[INFO] Note: SingleNamespace mode is configured via namespace: openshift-operators" >&2
        echo "[INFO]       with targetNamespaces: [tssc-tpa] in values.yaml" >&2
    else
        echo "[WARNING] values.yaml not found at $values_file, skipping subscription source/channel update" >&2
    fi
    
    echo "[INFO] ✓ Pre-release configuration complete" >&2
    echo "[INFO] tssc-cli will handle OperatorGroup and Subscription installation using values.yaml configuration" >&2
}

main() {
    parse_args "$@"
    
    configure_tpa
    echo "Done" >&2
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
fi
