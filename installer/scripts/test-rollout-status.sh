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
    RESOURCE_SELECTORS=()
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
            # The "rollout status" selectors, to find the actual resource to check for
            # successful rollout.
            RESOURCE_SELECTORS+=("$1")
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

rollout_status() {
    oc rollout status "${RESOURCE_TYPE}" \
        --namespace="${NAMESPACE}" \
        --watch \
        --timeout=10s \
        --selector="${1}"
}

assert_resource_exists() {
    local selector="${1}"
    local output
    output=$(
        oc get "${RESOURCE_TYPE}" \
            --namespace="${NAMESPACE}" \
            --selector="${selector}" 2>&1
    )
    local status=${?}
    if [[ $status -eq 0 && $output != "No resources found"* ]]; then
        info "Resource of type '${RESOURCE_TYPE}' with selector" \
            "'${selector}' exists in namespace '${NAMESPACE}'!"
        return 0
    fi

    warning "Resource of type '${RESOURCE_TYPE}' with selector" \
        "'${selector}' does not exist in namespace '${NAMESPACE}'."
    return 1
}

wait_for_resource() {
    for s in "${RESOURCE_SELECTORS[@]}"; do
        echo "# Checking if ${RESOURCE_TYPE} with selector '${s}' exists..."
        if ! assert_resource_exists "${s}"; then
            return 1
        fi

        echo "# Checking if ${RESOURCE_TYPE} with selector '${s}' is ready..."
        if ! rollout_status "${s}"; then
            echo -en "#\n# WARNING: ${RESOURCE_TYPE} '${s}' is not ready!\n#\n"
            return 1
        fi
        info "${RESOURCE_TYPE} objects with '${s}' selector are ready!"
    done
    return 0
}

test_rollout_status() {
    [[ -z "${NAMESPACE}" ]] && usage
    [[ -z "${RESOURCE_TYPE}" ]] && usage
    [[ ${#RESOURCE_SELECTORS[@]} -eq 0 ]] && usage

    for i in $(seq 1 "${RETRIES}"); do
        wait=$((i * 5))
        [[ $wait -gt 30  ]] && wait=30
        echo "### [${i}/${RETRIES}] Waiting for ${wait} seconds before retrying..."
        sleep ${wait}

        if wait_for_resource; then
            info "${RESOURCE_TYPE} objects ready: '${RESOURCE_SELECTORS[*]}'"
            return 0
        fi
    done

    fail "'${RESOURCE_TYPE}' are not ready!"
}

main() {
    parse_args "$@"

    # Namespace to check for "rollout status".
    declare -r NAMESPACE="${NAMESPACE:-}"
    # Resource type for "rollout status", as in "statefulset" or "deployment".
    declare -r RESOURCE_TYPE="${RESOURCE_TYPE:-statefulset}"
    # Number of retries to attempt before giving up.
    declare -r RETRIES=${RETRIES:-20}

    test_rollout_status
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
    echo
    echo "Success"
fi