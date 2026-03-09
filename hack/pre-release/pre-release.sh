#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR="$(
    cd "$(dirname "$0")" >/dev/null || exit
    pwd
)"

PROJECT_DIR="$(
    cd "$SCRIPT_DIR/../.." >/dev/null || exit
    pwd
)"

# Source common functions for configure_subscription
# shellcheck source=hack/pre-release/pre-release-common.sh
source "$SCRIPT_DIR/pre-release-common.sh"

usage() {
    echo "
Usage:
    ${0##*/} [options]

Optional arguments:
    -p, --product PRODUCT
        The product on which to activate the pre-release subscription.
        Can be specified multiple times.
        Supported products: rhdh, rhtas, rhtpa, gitops, pipelines
    -r, --tas-release-path PATH
        GitHub release path for TAS installation files.
        Example: https://github.com/securesign/releases/blob/release-1.3.1/1.3.1/stable
        Optional when using TAS product. If not provided, will auto-detect latest release
        using GitHub API (requires --github-token).
    -v, --tas-release-version VERSION
        TAS release version to use (e.g., 1.3.1).
        Defaults to \"latest\" which will fetch the most recent release.
        Requires --github-token for private repositories.
    -o, --tas-operator-version VERSION
        TAS operator CSV version to install (e.g., rhtas-operator.v1.3.2).
        Used to specify the exact operator version in the subscription.
        Can also be set via TAS_OPERATOR_VERSION environment variable.
    -t, --github-token TOKEN
        GitHub personal access token for accessing private repositories.
        Required for auto-detecting latest TAS release or accessing private repos.
        Can also be set via GITHUB_TOKEN environment variable.
    -d, --debug
        Activate tracing/debug mode.
    -h, --help
        Display this message.

Examples:
    # Auto-detect latest TAS release (requires GitHub token)
    ${0##*/} --product rhtas --github-token ghp_xxxxx
    
    # Specify TAS release version (requires GitHub token)
    ${0##*/} --product rhtas --tas-release-version 1.3.1 --github-token ghp_xxxxx
    
    # Use full TAS path (backward compatible)
    ${0##*/} --product rhtas --tas-release-path https://github.com/securesign/releases/blob/release-1.3.1/1.3.1/stable --github-token ghp_xxxxx
    
    # Configure RHDH
    ${0##*/} --product rhdh
    
    # Configure TPA
    ${0##*/} --product rhtpa
" >&2
}

parse_args() {
    PRODUCT_LIST=()
    TAS_RELEASE_PATH=""
    TAS_RELEASE_VERSION="latest"
    # Use GITHUB_TOKEN if set, otherwise fall back to GITOPS_GIT_TOKEN
    GITHUB_TOKEN="${GITHUB_TOKEN:-${GITOPS_GIT_TOKEN:-}}"
    while [[ $# -gt 0 ]]; do
        case $1 in
        -p|--product)
            case ${2:-} in
            developerHub|dh|rhdh)
                PRODUCT_LIST+=( "rhdh" )
                ;;
            gitops|pipelines)
                PRODUCT_LIST+=( "$2" )
                ;;
            trusted-artifact-signer|tas|rhtas)
                PRODUCT_LIST+=( "rhtas" )
                ;;
            trusted-profile-analyzer|tpa|rhtpa)
                PRODUCT_LIST+=( "rhtpa" )
                ;;
            "")
                echo "[ERROR] Product name needs to be specified after '--product'." >&2
                usage
                exit 1
                ;;
            *)
                echo "[ERROR] Unknown product: $2" >&2
                usage
                exit 1
                ;;
            esac
            shift
            ;;
        -r|--tas-release-path)
            if [[ -z "${2:-}" ]]; then
                echo "[ERROR] Release path needs to be specified after '--tas-release-path'." >&2
                usage
                exit 1
            fi
            TAS_RELEASE_PATH="$2"
            shift
            ;;
        -v|--tas-release-version)
            if [[ -z "${2:-}" ]]; then
                echo "[ERROR] Release version needs to be specified after '--tas-release-version'." >&2
                usage
                exit 1
            fi
            TAS_RELEASE_VERSION="$2"
            shift
            ;;
        -o|--tas-operator-version)
            if [[ -z "${2:-}" ]]; then
                echo "[ERROR] Operator version needs to be specified after '--tas-operator-version'." >&2
                usage
                exit 1
            fi
            TAS_OPERATOR_VERSION="$2"
            shift
            ;;
        -t|--github-token)
            if [[ -z "${2:-}" ]]; then
                echo "[ERROR] GitHub token needs to be specified after '--github-token'." >&2
                usage
                exit 1
            fi
            GITHUB_TOKEN="$2"
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

h1() {
    echo "
################################################################################
# $1
################################################################################
"
}

configure_gitops(){
    # GITOPS_IIB_IMAGE="quay.io/rhtap_qe/gitops-iib:782137"

    SUBSCRIPTION="openshiftGitOps"
    CHANNEL="latest"
    SOURCE="gitops-iib"
}

configure_pipelines(){
    # PIPELINES_IMAGE="quay.io/openshift-pipeline/openshift-pipelines-pipelines-operator-bundle-container-index"
    # PIPELINES_IMAGE_TAG="v4.17-candidate"

    SUBSCRIPTION="openshiftPipelines"
    CHANNEL="latest"
    SOURCE="pipelines-iib"
}

# configure_subscription is now in pre-release-common.sh

main() {
    parse_args "$@"
    
    # Validate TAS configuration if TAS is in product list
    if [[ " ${PRODUCT_LIST[*]} " =~ " rhtas " ]]; then
        if [[ -z "$TAS_RELEASE_PATH" ]] && [[ -z "${GITHUB_TOKEN:-}" ]]; then
            echo "[ERROR] Either --tas-release-path or --github-token is required when using TAS (rhtas) product" >&2
            echo "[ERROR] Option 1: Provide full path: --tas-release-path <path>" >&2
            echo "[ERROR] Option 2: Auto-detect latest: --github-token <token>" >&2
            echo "[ERROR] Option 3: Specify version: --tas-release-version <version> --github-token <token>" >&2
            usage
            exit 1
        fi
    fi
    
    for PRODUCT in $(echo "${PRODUCT_LIST[@]}" | tr " " "\n" | sort -u); do
        h1 "Configuring $PRODUCT"
        
        case "$PRODUCT" in
        rhdh)
            # Call product-specific script
            PRODUCT_SCRIPT="$SCRIPT_DIR/pre-release-rhdh.sh"
            if [[ ! -f "$PRODUCT_SCRIPT" ]]; then
                echo "[ERROR] Product script not found: $PRODUCT_SCRIPT" >&2
                exit 1
            fi
            SCRIPT_ARGS=()
            if [[ -n "${DEBUG:-}" ]]; then
                SCRIPT_ARGS+=("--debug")
            fi
            bash "$PRODUCT_SCRIPT" "${SCRIPT_ARGS[@]}"
            ;;
        rhtas)
            # Call product-specific script
            PRODUCT_SCRIPT="$SCRIPT_DIR/pre-release-tas.sh"
            if [[ ! -f "$PRODUCT_SCRIPT" ]]; then
                echo "[ERROR] Product script not found: $PRODUCT_SCRIPT" >&2
                exit 1
            fi
            SCRIPT_ARGS=()
            if [[ -n "$TAS_RELEASE_PATH" ]]; then
                SCRIPT_ARGS+=("--release-path" "$TAS_RELEASE_PATH")
            fi
            if [[ -n "$TAS_RELEASE_VERSION" && "$TAS_RELEASE_VERSION" != "latest" ]]; then
                SCRIPT_ARGS+=("--release-version" "$TAS_RELEASE_VERSION")
            fi
            if [[ -n "${TAS_OPERATOR_VERSION:-}" ]]; then
                SCRIPT_ARGS+=("--operator-version" "$TAS_OPERATOR_VERSION")
            fi
            if [[ -n "${GITHUB_TOKEN:-}" ]]; then
                SCRIPT_ARGS+=("--github-token" "$GITHUB_TOKEN")
            fi
            if [[ -n "${DEBUG:-}" ]]; then
                SCRIPT_ARGS+=("--debug")
            fi
            bash "$PRODUCT_SCRIPT" "${SCRIPT_ARGS[@]}"
            ;;
        rhtpa)
            # Call product-specific script
            PRODUCT_SCRIPT="$SCRIPT_DIR/pre-release-tpa.sh"
            if [[ ! -f "$PRODUCT_SCRIPT" ]]; then
                echo "[ERROR] Product script not found: $PRODUCT_SCRIPT" >&2
                exit 1
            fi
            SCRIPT_ARGS=()
            if [[ -n "${DEBUG:-}" ]]; then
                SCRIPT_ARGS+=("--debug")
            fi
            bash "$PRODUCT_SCRIPT" "${SCRIPT_ARGS[@]}"
            ;;
        gitops|pipelines)
            # These are simple configurations, keep them in main script
            "configure_$PRODUCT"
            if [[ -n "${SUBSCRIPTION:-}" ]]; then
                configure_subscription
            fi
            ;;
        *)
            echo "[ERROR] Unknown product: $PRODUCT" >&2
            exit 1
            ;;
        esac
        echo
    done
    echo "Updated subscription values.yaml"
    cat "$PROJECT_DIR/installer/charts/tssc-subscriptions/values.yaml"
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
fi
