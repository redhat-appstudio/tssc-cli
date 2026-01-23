#!/usr/bin/env bash
#
# Runs "oc rollout status" for configured namespace, resource type, and selectors.
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
    \$ export NAMESPACE=\"namespace\"
    \$ export RESOURCE_TYPE=\"deployment\"
    ${0##*/} <RESOURCE_SELECTORS>
" >&2
}

parse_args() {
    OPTIONS=()
    RESOURCES=()
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
        --**|-*)
            # Add options to the wait command
            OPTIONS+=("$1")
            ;;
        *)
            # The resource to wait for
            RESOURCES+=("$1")
            ;;
        esac
        shift
    done
}

fail() {
    echo "# [ERROR] ${*}" >&2
    exit 1
}

warning() {
    echo "# [WARNING] ${*}"
}

info() {
    echo "# [INFO] ${*}"
}

#
# Functions
#

wait_for_resources() {
    echo "# Checking conditions are met for resources..."
    if ! oc wait --timeout=10s "${OPTIONS[@]}" "${RESOURCES[@]}"; then
        echo -en "#\n# WARNING: not all resources are ready!\n#\n"
        return 1
    fi
    info "All resources are ready!"
    return 0
}

test_wait_for() {
    [[ ${#RESOURCES[@]} -eq 0 ]] && usage

    for i in $(seq 1 "${RETRIES}"); do
        wait=$((i * 5))
        [[ $wait -gt 30  ]] && wait=30
        echo "### [${i}/${RETRIES}] Waiting for ${wait} seconds before retrying..."
        sleep ${wait}

        if wait_for_resources; then
            return 0
        fi
    done

    fail "Not all resources are ready after ${RETRIES} retries!"
}

main() {
    parse_args "$@"

    # Number of retries to attempt before giving up.
    declare -r RETRIES=${RETRIES:-20}

    test_wait_for
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
    echo
    echo "Success"
fi