#!/usr/bin/env bash
#
# Tests if the ArgoCD instance is available on the cluster by logging in.
#
# Uses the ArgoCD session, created by previously running "argocd login", to
# generate an account token. The information is then stored in a kubernetes
# secret.
#

shopt -s inherit_errexit
set -Eeu -o pipefail

declare -r SUBCOMMAND="${1:-}"

# ArgoCD hostname (FQDN) to test.
declare -r ARGOCD_HOSTNAME="${ARGOCD_HOSTNAME:-}"
# ArgoCD username to use for login.
declare -r ARGOCD_USER="${ARGOCD_USER:-admin}"
# ArgoCD password to use for login.
declare -r ARGOCD_PASSWORD="${ARGOCD_PASSWORD:-}"
# Environment file to store the ArgoCD credentials.
declare -r ARGOCD_ENV_FILE="${ARGOCD_ENV_FILE:-/tssc/argocd/env}"
# Target secret name, to be created with ArgoCD credentials.
declare -r SECRET_NAME="${SECRET_NAME:-tssc-argocd-integration}"
# Secret's namespace.
declare -r NAMESPACE="${NAMESPACE:-}"

fail() {
    echo "# [ERROR] ${*}" >&2
    exit 1
}

# Asserts the required environment variables.
assert_variables() {
    case "${SUBCOMMAND}" in
        login | generate)
            [[ -z "${ARGOCD_HOSTNAME}" ]] &&
                fail "ARGOCD_HOSTNAME is not set!"
            [[ -z "${ARGOCD_USER}" ]] &&
                fail "ARGOCD_USER is not set!"
            [[ -z "${ARGOCD_PASSWORD}" ]] &&
                fail "ARGOCD_PASSWORD is not set!"
            ;;
        store)
            [[ -z "${NAMESPACE}" ]] &&
                fail "NAMESPACE is not set!"
            [[ -z "${SECRET_NAME}" ]] &&
                fail "SECRET_NAME is not set!"
            ;;
    esac
}

# Executes the ArgoCD login command.
argocd_login() {
    argocd login "${ARGOCD_HOSTNAME}" \
        --grpc-web \
        --insecure \
        --skip-test-tls \
        --http-retry-max="5" \
        --username="${ARGOCD_USER}" \
        --password="${ARGOCD_PASSWORD}"
}

# Retries a few times until the ArgoCD instance is available.
test_argocd_login() {
    echo "# Logging into ArgoCD on '${ARGOCD_HOSTNAME}'..."
    for i in {1..30}; do
        wait=$((i * 5))
        echo "### [${i}/30] Waiting for ${wait} seconds before retrying..."
        sleep ${wait}

        echo "# [${i}/30] Testing ArgoCD login on '${ARGOCD_HOSTNAME}'..."
        argocd_login &&
            return 0
    done
    return 1
}

# Generates the ArgoCD API token.
argocd_generate_token() {
    echo "# Generating ArgoCD API token on '${ARGOCD_HOSTNAME}'..."
    ARGOCD_API_TOKEN="$(
        argocd account generate-token \
            --grpc-web \
            --insecure \
            --http-retry-max="5" \
            --account="${ARGOCD_USER}"
    )" || fail "ArgoCD API token could not be generated!"
    if [[ "${?}" -ne 0 || -z "${ARGOCD_API_TOKEN}" ]]; then
        fail "ArgoCD API token could not be generated!"
    fi

    echo "# Storing ArgoCD API credentials in '${ARGOCD_ENV_FILE}'..."
    cat <<EOF >"${ARGOCD_ENV_FILE}" || fail "Fail to write '${ARGOCD_ENV_FILE}'!"
ARGOCD_HOSTNAME=${ARGOCD_HOSTNAME}
ARGOCD_USER=${ARGOCD_USER}
ARGOCD_PASSWORD=${ARGOCD_PASSWORD}
ARGOCD_API_TOKEN=${ARGOCD_API_TOKEN}
EOF

    return 0
}

# Waits for the environment file to be available.
wait_for_env_file() {
    echo "# Waiting for '${ARGOCD_ENV_FILE}' to be available..."
    for i in {1..30}; do
        wait=$((i * 5))
        echo "### [${i}/30] Waiting for '${ARGOCD_ENV_FILE}' to be available..."
        sleep ${wait}

        [[ -r "${ARGOCD_ENV_FILE}" ]] &&
            return 0
    done
    return 1
}

# Stores the ArgoCD credentials in a Kubernetes secret.
argocd_store_credentials() {
    # Using the dry-run flag to generate the secret payload, and later on "kubectl
    # apply" to create, or update, the secret payload in the cluster.
    echo "# Creating secret '${SECRET_NAME}' in namespace '${NAMESPACE}' from '${ARGOCD_ENV_FILE}'..."
    if ! (
        kubectl create secret generic "${SECRET_NAME}" \
            --namespace="${NAMESPACE}" \
            --from-env-file="${ARGOCD_ENV_FILE}" \
            --dry-run="client" \
            --output="yaml" |
            kubectl apply -f -
    ); then
        fail "Secret '${SECRET_NAME}' could not be created!"
    fi
    return 0
}

#
# Main
#

assert_variables &&
    echo "# All environment variables are set"

case "${SUBCOMMAND}" in
login)
    if test_argocd_login; then
        echo "# ArgoCD is available: '${ARGOCD_HOSTNAME}'"
        exit 0
    else
        fail "ArgoCD not available!"
    fi
    ;;
generate)
    test_argocd_login ||
        fail "ArgoCD not available!"

    if argocd_generate_token; then
        echo "# ArgoCD API token generated successfully!"
        exit 0
    else
        fail "ArgoCD API token could not be generated!"
    fi
    ;;
store)
    wait_for_env_file ||
        fail "ARGOCD_ENV_FILE='${ARGOCD_ENV_FILE}' not found or not readable!"

    if argocd_store_credentials; then
        echo "# ArgoCD API credentials stored successfully!"
        exit 0
    else
        fail "ArgoCD API credentials could not be stored!"
    fi
    ;;
*)
    fail "Invalid subcommand provided: '${SUBCOMMAND}'. " \
        "Use 'login', 'generate' or 'store'!"
    ;;
esac
