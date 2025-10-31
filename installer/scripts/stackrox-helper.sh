#!/usr/bin/env bash
#
# Generates a StackRox API token and stores it on a Kubernetes secret.
#
#   https://access.redhat.com/solutions/5907651
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
            info "Running script as: $(id)"
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

assert_variables() {

    # StackRox API username.
    ROX_USERNAME="${ROX_USERNAME:-admin}"
    # StackRox API password.
    ROX_PASSWORD="${ROX_PASSWORD:-}"
    # StackRox API base endpoint.
    ROX_ENDPOINT="${ROX_ENDPOINT:-}"
    # StackRox API endpoint path to generate a token.
    ROX_ENDPOINT_PATH="${ROX_ENDPOINT_PATH:-/v1/apitokens/generate}"

    # Kubernetes secret namespace and name to store the generated token.
    NAMESPACE="${NAMESPACE:-}"
    SECRET_NAME="${SECRET_NAME:-tssc-acs-integration}"

    [[ -n "${ROX_USERNAME}" ]] ||
        fail "ROX_USERNAME is not set!"
    [[ -n "${ROX_PASSWORD}" ]] ||
        fail "ROX_PASSWORD is not set!"
    [[ -n "${ROX_ENDPOINT}" ]] ||
        fail "ROX_ENDPOINT is not set!"
    [[ -n "${SECRET_NAME}" ]] ||
        fail "SECRET_NAME is not set!"
    [[ -n "${NAMESPACE}" ]] ||
        fail "NAMESPACE is not set!"
}

# Stores the new token and API endpoint in a secret.
store_api_token_in_secret() {
    info "# Storing StackRox API token on secret '${NAMESPACE}/${SECRET_NAME}'..."
    declare -r token="${1:-}"
    [[ -z "${token}" ]] &&
        fail "Token is not informed!"

    if ! oc create secret generic "${SECRET_NAME}" \
            --namespace="${NAMESPACE}" \
            --from-literal="endpoint=${ROX_ENDPOINT}:443" \
            --from-literal="token=${token}" \
            --dry-run="client" \
            --output="yaml" |
            kubectl apply -f -; then
        fail "Failed to store StackRox API token in a secret."
    fi
    info "Token stored successfully."
}

# Generates a StackRox API token and stores it as a Kubernetes secret.
stackrox_generate_api_token() {
    api_url="https://${ROX_ENDPOINT}${ROX_ENDPOINT_PATH}"
    info "# Generating StackRox API token on ${api_url}" \
        "for user '${ROX_USERNAME}'..."
    output="$(
        curl \
            --silent \
            --insecure \
            --user "${ROX_USERNAME}:${ROX_PASSWORD}" \
            --data '{"name":"TSSC", "role": "Admin"}' \
            "${api_url}"
    )"
    [[ $? -ne 0 || -z "${output}" ]] &&
        fail "Failed to generate StackRox API token."

    token="$(echo "${output}" | jq -r '.token')"
    [[ -z "${token}" ]] &&
        fail "Failed to extract StackRox API token."
    info "Token generated successfully."
}

#
# Main
#
main() {
    parse_args "$@"

    assert_variables
    stackrox_generate_api_token
    store_api_token_in_secret "${token}"
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
    echo
    echo "Success"
fi
