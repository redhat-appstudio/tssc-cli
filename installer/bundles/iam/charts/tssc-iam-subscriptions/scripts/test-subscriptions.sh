#!/usr/bin/env bash
#
# Tests if the requested CRDs are available on the cluster.
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
    CRDS=()
    while [[ $# -gt 0 ]]; do
        case "$1" in
        -d | --debug)
            set -x
            DEBUG="--debug"
            export DEBUG
            info "Running script as: $(id)"
            ;;
        -h | --help)
            usage
            exit 0
            ;;
        *)
            CRDS+=("$1")
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

# Tests if the CRDs are available on the cluster, returns true when all CRDs are
# found, otherwise false.
api_resources_available() {
    SUCCESS=0
    for crd in "${CRDS[@]}"; do
        if oc get customresourcedefinitions "${crd}" >/dev/null 2>&1; then
            echo "# CRD '${crd}' is installed."
        else
            echo "# ERROR: CRD '${crd}' not visible to this Pod (missing or RBAC)."
            oc get customresourcedefinitions "${crd}" 2>&1 | sed 's/^/#   oc: /' || true
            SUCCESS=1
        fi
    done
    return "$SUCCESS"
}

# Verifies the availability of the CRDs, retrying a few times.
test_subscriptions() {
    if [[ ${#CRDS[@]} -eq 0 ]]; then
        echo "# No operator Subscriptions are Helm-managed for this release; skipping CRD wait."
        return 0
    fi

    echo "# Waiting for CRDs to be available: '${CRDS[*]}'"
    for i in {1..40}; do
        echo "# Check ${i}/40"
        if api_resources_available; then
            info "# CRDs are available: '${CRDS[*]}'"
            return 0
        fi
        wait=$((i * 3))
        if [[ ${wait} -gt 60 ]]; then wait=60; fi
        echo "# Waiting for ${wait} seconds before retrying..."
        sleep ${wait}
    done
    fail "CRDs not available!"
}

#
# Main
#
main() {
    parse_args "$@"
    test_subscriptions
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
    echo
    echo "Success"
fi
