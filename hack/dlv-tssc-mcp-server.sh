#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Script's directory full path.
declare SCRIPT_DIR
SCRIPT_DIR="$(dirname "${0}")"
# Installer executable, by default relative to this script.
declare TSSC_BIN="${SCRIPT_DIR}/../bin/tssc"

# Installer container image, a required flag for the mcp-server subcommand.
declare TSSC_IMAGE="ghcr.io/redhat-appstudio/tssc:latest"
# Port to expose DLV API.
declare DLV_PORT="8282"

# Shows the script usage and flags.
usage() {
    echo "
This script wraps 'dlv exec' on 'tssc mcp-server', preserves the STDIO
communication while exposing Delve's API on a local port. The goal is using this
script as the command for the agentic LLM client while running Delve debugger
against the CLI.

Usage:
    ${0##*/} [options]

Optional arguments:
    -i, --image IMAGE
        The installer container image to use for 'tssc mcp-server'.
        Default: ${TSSC_IMAGE}
    -p, --port PORT
        The port to expose Delve API.
        Default: ${DLV_PORT}
    -d, --debug
        Enable debug information.
    -h, --help
        Display this message.
" >&2
}

# Parse command-line flags.
parse_args() {
    while [[ ${#} -gt 0 ]]; do
        case ${1} in
        -d | --debug)
            set -x
            ;;
        -i | --image)
            TSSC_IMAGE="${2}"
            shift
            ;;
        -p | --port)
            DLV_PORT="${2}"
            shift
            ;;
        -h | --help)
            usage
            exit 0
            ;;
        *)
            echo "[ERROR] Unknown argument: '${1}'" >&2
            usage
            exit 1
            ;;
        esac
        shift
    done
}

# Runs the Delve debugger for the TSSC MCP server.
main() {
    parse_args "${@}"

    if ! which dlv >/dev/null 2>&1; then
        echo "[ERROR] Delve (dlv) is not installed. Please install it first." >&2
        exit 1
    fi

    if [ ! -f "${TSSC_BIN}" ]; then
        echo "ERROR: Can't find the executable at '${TSSC_BIN}'!" >&2
        exit 1
    fi

    # Running "tssc mcp-server" ensuring the log output is suppressed.
    exec dlv exec \
        --headless \
        --listen=":${DLV_PORT}" \
        --log \
        --log-dest="/dev/null" \
        "${TSSC_BIN}" -- mcp-server --image="${TSSC_IMAGE}"
}

main "${@}"
