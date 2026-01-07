#!/usr/bin/env bash

# shellcheck disable=SC2016

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR="$(
    cd "$(dirname "$0")" >/dev/null
    pwd
)"

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
    PROJECT_DIR="$(
        cd "$(dirname "$SCRIPT_DIR")" >/dev/null
        pwd
    )"
    while [[ $# -gt 0 ]]; do
        case $1 in
        -d | --debug)
            set -x
            DEBUG="--debug"
            export DEBUG
            ;;
        -h | --help)
            usage
            exit 0
            ;;
        *)
            echo "[ERROR] Unknown argument: $1"
            usage
            exit 1
            ;;
        esac
        shift
    done
}

init() {
    cd "$PROJECT_DIR"
    CONFIG="$PROJECT_DIR/hack/config/config.yaml"
}

h1() {
    echo
    echo "# $1 ##########################################################################" | cut -c -80
}

getNamespace() {
    export PRODUCT="$1"
    NAMESPACE="$(yq '.tssc.products[] | select(.name == strenv(PRODUCT)) | .namespace' "$CONFIG")"
    oc get namespace "$NAMESPACE" > /dev/null 2>&1
    return $?
}

tas() {
    if ! getNamespace "Trusted Artifact Signer"; then
        return
    fi
    h1 "Trusted Artifact Signer"
    ROUTE="$(oc get route -n $NAMESPACE --selector="app.kubernetes.io/name=rekor-search-ui" -o yaml | yq '.items[0].metadata.name')"
    echo "  - URL: $(oc get route -n "$NAMESPACE" "$ROUTE" -o jsonpath="https://{.spec.host}")"
}

tpa() {
    if ! getNamespace "Trusted Profile Analyzer"; then
        return
    fi
    h1 "Trusted Profile Analyzer"
    ROUTE="$(oc get route -n $NAMESPACE --selector="app.kubernetes.io/name=server" -o yaml | yq '.items[0].metadata.name')"
    echo "  - URL: $(oc get route -n "$NAMESPACE" "$ROUTE" -o jsonpath="https://{.spec.host}")"
}

tssc() {
    NAMESPACE="tssc"
    h1 "TSSC"
    echo "  - User: admin"
    echo "  - Password: $(oc get secret -n "$NAMESPACE" tssc-realms-admin-user -o jsonpath="{.data.password}" | base64 -d)"
}

action() {
    tas
    tpa
    tssc
}

main() {
    parse_args "$@"
    init
    action
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
fi
