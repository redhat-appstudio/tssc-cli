#!/usr/bin/env bash
#
# Tests if the given KeycloakRealmImports are imported without errors.
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
    KEYCLOAKREALMIMPORT_NAMES=()
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
            # List of KeycloakRealmImports to test.
            KEYCLOAKREALMIMPORT_NAMES+=("$1")
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

keycloakrealmimport_available() {
    for r in "${KEYCLOAKREALMIMPORT_NAMES[@]}"; do
        info "Checking if KeycloakRealmImport '${r}' has errors..."
        if ! oc get keycloakrealmimports "${r}" \
                --namespace="${NAMESPACE}" &>/dev/null; then
            echo "# [ERROR] KeycloakRealmImport '${r}' not found!"
            return 1
        fi

        has_errors="$(
            oc get keycloakrealmimports "${r}" \
                --namespace="${NAMESPACE}" \
                --output=jsonpath='{.status.conditions[?(@.type=="HasErrors")].status}'
        )"
        info "KeycloakRealmImport '${r}' condition='HasErrors=${has_errors}'"

        if [[ "${has_errors}" == "True" ]]; then
            return 1
        fi
    done
    return 0
}

test_keycloakrealmimport() {
    if [[ -z "${NAMESPACE}" ]]; then
        fail "Usage: NAMESPACE=namespace $0 <STATEFULSETS>"
    fi

    if [[ ${#KEYCLOAKREALMIMPORT_NAMES[@]} -eq 0 ]]; then
        fail "Usage: $0 <KEYCLOAKREALMIMPORT_NAMES>"
    fi

    for i in {1..10}; do
        if keycloakrealmimport_available; then
            info "KeycloakRealmImports are available: '${KEYCLOAKREALMIMPORT_NAMES[*]}'"
            return 0
        fi
        wait=$((i * 1))
        echo -e "### [${i}/10] Waiting for ${wait} seconds before retrying...\n"
        sleep ${wait}
    done
    fail "KeycloakRealmImports not available!"
}

main() {
    parse_args "$@"

    # Namespace to check for KeycloakRealmImports.
    declare -r NAMESPACE="${NAMESPACE:-}"

    test_keycloakrealmimport
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
    echo
    echo "Success"
fi
