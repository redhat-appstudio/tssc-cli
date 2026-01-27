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

# TODO: Investigate how to get latest RHDH release tag
# The RHDH install script is at: https://raw.githubusercontent.com/redhat-developer/rhdh-operator/main/.rhdh/scripts/install-rhdh-catalog-source.sh
# Potential approaches:
# 1. Query GitHub API for latest release in redhat-developer/rhdh-operator repository
# 2. Query the install script itself to see if it has version information
# 3. Check if the script accepts a version parameter
# For now, we use --latest flag which the script handles internally
fetch_latest_rhdh_release() {
    local owner="redhat-developer"
    local repo="rhdh-operator"
    local api_url="https://api.github.com/repos/${owner}/${repo}/releases/latest"
    
    if [[ -z "${GITHUB_TOKEN:-}" ]]; then
        echo "[INFO] No GitHub token provided, install script will use --latest flag" >&2
        return 0
    fi
    
    echo "[INFO] Fetching latest RHDH release from GitHub API..." >&2
    if [[ -n "${DEBUG:-}" ]]; then
        echo "[DEBUG] API URL: $api_url" >&2
    fi
    
    local curl_args=(-sSLf)
    curl_args+=(-H "Authorization: token ${GITHUB_TOKEN}")
    curl_args+=(-H "Accept: application/vnd.github.v3+json")
    
    local response
    if ! response=$(curl "${curl_args[@]}" "$api_url" 2>&1); then
        echo "[WARNING] Failed to fetch latest release from GitHub API, install script will use --latest flag" >&2
        return 0
    fi
    
    # Extract the tag_name from the latest release response
    local latest_tag
    latest_tag=$(echo "$response" | grep -o '"tag_name":"[^"]*"' | head -n1 | cut -d'"' -f4)
    
    if [[ -z "$latest_tag" ]]; then
        echo "[WARNING] Failed to parse latest release tag, install script will use --latest flag" >&2
        return 0
    fi
    
    echo "[INFO] Found latest RHDH release tag: $latest_tag" >&2
    echo "$latest_tag"
}

configure_rhdh() {
    echo "[INFO] Configuring RHDH (Red Hat Developer Hub) Operator for pre-release testing" >&2
    
    # TODO: Use the latest release tag if we can fetch it
    # For now, the install script handles --latest internally
    local latest_tag
    latest_tag=$(fetch_latest_rhdh_release)
    
    RHDH_INSTALL_SCRIPT="https://raw.githubusercontent.com/redhat-developer/rhdh-operator/main/.rhdh/scripts/install-rhdh-catalog-source.sh"
    echo "[INFO] Downloading RHDH install script..." >&2
    curl -sSLO "$RHDH_INSTALL_SCRIPT"
    chmod +x install-rhdh-catalog-source.sh

    echo "[INFO] Running RHDH install script with --latest flag..." >&2
    ./install-rhdh-catalog-source.sh --latest --install-operator rhdh

    SUBSCRIPTION="developerHub"
    CHANNEL="fast-1.9"
    SOURCE="rhdh-fast"
    export SUBSCRIPTION
    export CHANNEL
    export SOURCE
}

# configure_subscription is now in pre-release-common.sh

main() {
    parse_args "$@"
    
    configure_rhdh
    # Only configure subscription if SUBSCRIPTION variable is set
    if [[ -n "${SUBSCRIPTION:-}" ]]; then
        configure_subscription
    fi
    echo "Done" >&2
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
fi
