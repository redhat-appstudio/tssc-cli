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

# ensure_umoci makes umoci available on PATH by using the system copy or
# downloading a prebuilt binary from GitHub releases (no base image change needed for CI).
ensure_umoci() {
    if command -v umoci &>/dev/null; then
        echo "[INFO] Using existing umoci: $(command -v umoci)" >&2
        return 0
    fi
    local os arch asset_name url tmpdir
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)
    case "$arch" in
        x86_64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *)
            echo "[ERROR] Unsupported architecture for umoci prebuilt: $arch (only amd64 and arm64 are supported)" >&2
            return 1
            ;;
    esac
    asset_name="umoci.${os}.${arch}"
    url="https://github.com/opencontainers/umoci/releases/download/v0.6.0/${asset_name}"
    tmpdir=$(mktemp -d)
    echo "[INFO] Downloading umoci prebuilt from GitHub releases into $tmpdir..." >&2
    if ! curl -sSLf -o "$tmpdir/umoci" "$url"; then
        echo "[ERROR] Failed to download umoci from $url" >&2
        rm -rf "$tmpdir"
        return 1
    fi
    chmod +x "$tmpdir/umoci"
    export PATH="$tmpdir:$PATH"
    echo "[INFO] umoci is now available on PATH (from $tmpdir)" >&2
}

# ensure_opm makes opm v1.47+ available on PATH by using the system copy or
# downloading a prebuilt binary from operator-registry releases (no base image change needed for CI).
ensure_opm() {
    if command -v opm &>/dev/null; then
        echo "[INFO] Using existing opm: $(command -v opm)" >&2
        return 0
    fi
    local os arch asset_name url tmpdir
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)
    case "$arch" in
        x86_64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *)
            echo "[ERROR] Unsupported architecture for opm prebuilt: $arch (only amd64 and arm64 are supported)" >&2
            return 1
            ;;
    esac
    asset_name="${os}-${arch}-opm"
    url="https://github.com/operator-framework/operator-registry/releases/download/v1.63.0/${asset_name}"
    tmpdir=$(mktemp -d)
    echo "[INFO] Downloading opm prebuilt from operator-registry releases into $tmpdir..." >&2
    if ! curl -sSLf -o "$tmpdir/opm" "$url"; then
        echo "[ERROR] Failed to download opm from $url" >&2
        rm -rf "$tmpdir"
        return 1
    fi
    chmod +x "$tmpdir/opm"
    export PATH="$tmpdir:$PATH"
    echo "[INFO] opm is now available on PATH (from $tmpdir)" >&2
}

configure_rhdh() {
    echo "[INFO] Configuring RHDH (Red Hat Developer Hub) Operator for pre-release testing" >&2

    ensure_umoci || exit 1
    ensure_opm || exit 1

    RHDH_INSTALL_SCRIPT="https://raw.githubusercontent.com/redhat-developer/rhdh-operator/main/.rhdh/scripts/install-rhdh-catalog-source.sh"
    echo "[INFO] Downloading RHDH install script..." >&2
    curl -sSLO "$RHDH_INSTALL_SCRIPT"
    chmod +x install-rhdh-catalog-source.sh

    echo "[INFO] Running RHDH install script with --latest flag..." >&2
    local max_attempts=3
    local wait_seconds=120
    local attempt=1
    while true; do
        if ./install-rhdh-catalog-source.sh --latest; then
            break
        fi
        if [[ $attempt -ge $max_attempts ]]; then
            echo "[ERROR] RHDH install script failed after $max_attempts attempts (e.g. registry 500). Try again later." >&2
            exit 1
        fi
        echo "[WARNING] RHDH install script failed (attempt $attempt/$max_attempts). Waiting ${wait_seconds}s before retry..." >&2
        sleep "$wait_seconds"
        attempt=$((attempt + 1))
    done

}

# Subscription values for RHDH (used by configure_subscription in pre-release-common.sh)
SUBSCRIPTION="developerHub"
CHANNEL="fast-1.9"
SOURCE="rhdh-fast"
export SUBSCRIPTION CHANNEL SOURCE

main() {
    parse_args "$@"

    configure_rhdh
    # Always update installer values.yaml for RHDH subscription (vars set above)
    configure_subscription
    echo "Done" >&2
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
fi
