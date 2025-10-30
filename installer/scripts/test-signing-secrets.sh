#!/usr/bin/env bash
#
# Check that the signing secret is not empty.
#

shopt -s inherit_errexit
set -o errexit
set -o errtrace
set -o nounset
set -o pipefail

usage() {
    echo "
Usage:
    ${0##*/} [options]

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
    # Number of retries to attempt before giving up.
    RETRIES=${RETRIES:-20}
    while [[ $# -gt 0 ]]; do
        case $1 in
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

status() {
    items=$(
        oc get secret \
            --ignore-not-found \
            --namespace="openshift-pipelines" \
            --output=jsonpath="{.data}" \
        "signing-secrets" | sed 's:",":\n:g' | grep -c '":"'
    )
    if [ "$items" -lt 3 ]; then
        return 1
    fi
    return 0
}

test_signing_secrets() {
    for i in $(seq 0 "${RETRIES}"); do
        wait=$(( i * 5))
        [[ $wait -gt 30  ]] && wait=30
        info "[${i}/${RETRIES}] Waiting for ${wait} seconds before retrying..."
        sleep ${wait}

        if status; then
            info "signing-secrets ready"
            return 0
        fi
    done
    fail "signing-secrets not ready!"
}

#
# Main
#
main() {
    parse_args "$@"
    test_signing_secrets
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
    echo
    echo "Success"
fi
