#!/usr/bin/env bash
#
# Test whether the informed URL is online, and returning the expected status code.
#

shopt -s inherit_errexit
set -o errexit
set -o errtrace
set -o nounset
set -o pipefail

usage() {
    echo "
Usage:
    ${0##*/}

Optional arguments:
    -d, --debug
        Activate tracing/debug mode.
    -h, --help
        Display this message.

Example:
    ${0##*/}
" >&2
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
        -d | --debug)
            set -x
            DEBUG="--debug"
            export DEBUG
            echo "Running script as: $(id)"
            ;;
        -h | --help)
            usage
            exit 0
            ;;
        *)
            fail "Unsupported argument: '$1'."
            ;;
        esac
        shift
    done
}

fail() {
    echo "# [ERROR] ${*}" >&2
    exit 1
}

info() {
    echo "# [INFO] ${*}"
}

#
# Functions
#

# Tests if the URL is online and returns the expected HTTP status code.
probe_url() {
    local response_code
    local curl_exit

    info "# Probing URL '${URL}' for the status code '${STATUS_CODE}'... "

    # Fetch the HTTP status code from the URL.
    response_code=$(
        curl \
            --silent \
            --show-error \
            --fail \
            --location \
            --insecure \
            --max-time 30 \
            --output /dev/null \
            --write-out "%{http_code}" \
            "${URL}"
    ) || curl_exit=${?}
    
    if [[ "${curl_exit:-0}" -ne 0 ]]; then
        echo "# ERROR: Failed to fetch URL '${URL}', returned '${curl_exit}'." >&2
        return 1
    fi

    if [[ "${response_code}" -eq "${STATUS_CODE}" ]]; then
        echo "# INFO: URL '${URL}' is online and returned '${response_code}'."
        return 0
    else
        echo "# ERROR: '${URL}' returned status code '${response_code}'" \
            " expected ${STATUS_CODE}." >&2
        return 1
    fi
}

test_url() {
    if [[ -z "${URL}" ]]; then
        echo "# ERROR: URL environment variable is not set." >&2
        exit 1
    fi

    if [[ -z "${STATUS_CODE}" ]]; then
        echo "# ERROR: STATUS_CODE environment variable is not set." >&2
        exit 1
    fi

    # Probe the URL until it returns the expected HTTP status code, or exceeds the
    # retry limit. Each retry waits for a multiple of the previous retry interval.
    for i in {1..15}; do
        if probe_url; then
            info "# SUCCESS: URL '${URL}' returned expected status code '${STATUS_CODE}'."
            return 0
        fi
        wait=$((i * 3))
        echo -e "# WARN: [${i}/15] Waiting for ${wait}s before retrying...\n"
        sleep ${wait}
    done
    fail "URL '${URL}' is not accessible or returned an unexpected status code."
}

#
# Main
#
main() {
    parse_args "$@"

    # Target URL to test.
    declare -r URL="${URL:-}"
    # Expected HTTP status code. Default to 200.
    declare -r STATUS_CODE="${STATUS_CODE:-200}"

    test_url
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
    echo
    echo "Success"
fi
