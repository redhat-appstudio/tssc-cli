#!/usr/bin/env bash
#
# Script to initialize the Quay super-user and create a "docker-registry" secret
# with the admin credentials, the secret is enhanced with the "token" and "url"
# needed for later integration with Red Hat Developer Hub.
#
# The script also initializes a new organization using the super-user token
# obtained. For the new organization it needs a name and a different email than
# the super-user's.
#
#

shopt -s inherit_errexit
set -Eeu -o pipefail
set -x

# Quay hostname (FQDN).
declare -r QUAY_HOSTNAME="${QUAY_HOSTNAME:-}"

# Quay super-user credentials.
declare -r QUAY_USERNAME="${QUAY_USERNAME:-admin}"
declare -r QUAY_PASSWORD="${QUAY_PASSWORD:-}"
declare -r QUAY_EMAIL="${QUAY_EMAIL:-admin@localhost}"

# Quay organization name to create and email address.
declare -r QUAY_ORGANIZATION="${QUAY_ORGANIZATION:-tssc}"
declare -r QUAY_ORGANIZATION_EMAIL="${QUAY_ORGANIZATION_EMAIL:-${QUAY_ORGANIZATION}@localhost}"

# Namespace and secret name to create the "docker-registry" secret.
declare -r NAMESPACE="${NAMESPACE:-}"
declare -r SECRET_NAME="${SECRET_NAME:-}"

# After initialization of the super-user, or login, the global variable will be
# set with the user's access token obtained from Quay.
declare ACCESS_TOKEN=""

# Quay robot account for register
declare QUAY_ROBOT_SHORT_NAME="${QUAY_ROBOT_SHORT_NAME:-tssc_rw}"
declare QUAY_ROBOT_USERNAME=""
declare QUAY_ROBOT_TOKEN=""

# Quay read only reobot account
declare QUAY_ROBOT_SHORT_NAME_READONLY="${QUAY_ROBOT_SHORT_NAME_READONLY:-tssc_ro}"
declare QUAY_ROBOT_USERNAME_READONLY=""
declare QUAY_ROBOT_TOKEN_READONLY=""

#
# Functions
#

fail() {
    echo "# [ERROR] ${*}" >&2
    exit 1
}

warn() {
    echo "# [WARN] ${*}" >&2
}

info() {
    echo "# [INFO] ${*}"
}

assert_variables() {
    [[ -z "${QUAY_HOSTNAME}" ]] &&
        fail "QUAY_HOSTNAME is not set!"
    [[ -z "${QUAY_USERNAME}" ]] &&
        fail "QUAY_USERNAME is not set!"
    [[ -z "${QUAY_PASSWORD}" ]] &&
        fail "QUAY_PASSWORD is not set!"
    [[ -z "${QUAY_EMAIL}" ]] &&
        fail "QUAY_EMAIL is not set!"
    [[ -z "${QUAY_ORGANIZATION}" ]] &&
        fail "QUAY_ORGANIZATION is not set!"
    [[ -z "${QUAY_ORGANIZATION_EMAIL}" ]] &&
        fail "QUAY_ORGANIZATION_EMAIL is not set!"

    [[ "${QUAY_EMAIL}" == "${QUAY_ORGANIZATION_EMAIL}" ]] &&
        fail "QUAY_EMAIL and QUAY_ORGANIZATION_EMAIL must be different!"

    [[ -z "${NAMESPACE}" ]] &&
        fail "NAMESPACE is not set!"
    [[ -z "${SECRET_NAME}" ]] &&
        fail "SECRET_NAME is not set!"
}

# Initializes the Quay super-user with the credentials informed via environment.
quay_initialize_super_user() {
    local quay_url="https://${QUAY_HOSTNAME}/api/v1/user/initialize"
    local data=(
        "{"
        "\"username\": \"${QUAY_USERNAME}\","
        "\"password\": \"${QUAY_PASSWORD}\","
        "\"email\": \"${QUAY_EMAIL}\","
        "\"access_token\": true"
        "}"
    )
    local init_response token message

    info "Initializing Quay super-user..."
    init_response=$(
        curl \
            --silent \
            --insecure \
            --request POST \
            --header 'Content-Type: application/json' \
            --data "${data[*]}" \
            "${quay_url}"
    )
    message=$(echo "${init_response}" | jq --raw-output '.message')

    info "Inspecting Quay initialization response..."
    # Checking the response message to determine if Quay has already been
    # initialized.
    if [[ "${message}" == *"a non-empty database"* ]]; then
        warn "Quay has already been initialized!"
        return 0
    fi

    info "Extracting access token from the response..."
    # When the super user has not been initialized and the response doesn't
    # contain the expected "access_token", the script should fail completely.
    token=$(echo "${init_response}" | jq --raw-output '.access_token')
    if [[ -z "${token}" || "${token}" == "null" ]]; then
        fail "Failed to initialize Quay super-user!"
    fi

    info "Quay access token obtained successfully!"
    export ACCESS_TOKEN="${token}"
    return 0
}

# Creates a new organization in Quay with the name informed via environment, using
# the super-user's ACCESS_TOKEN to authorize the request.
quay_create_organization() {
    local quay_url="https://${QUAY_HOSTNAME}/api/v1/organization"
    local data=(
        "{"
        "\"name\": \"${QUAY_ORGANIZATION}\","
        "\"email\": \"${QUAY_ORGANIZATION_EMAIL}\""
        "}"
    )
    local create_response

    info "Creating Quay organization '${QUAY_ORGANIZATION}'..."
    create_response=$(
        curl \
            --silent \
            --insecure \
            --location \
            --request POST \
            --header 'Content-Type: application/json' \
            --header "Authorization: Bearer ${ACCESS_TOKEN}" \
            --data "${data[*]}" \
            "${quay_url}"
    )

    if [[ -z "${create_response}" || (${create_response} != *"Created"* && ${create_response} != *"already exists"*) ]]; then
        fail "Failed to create organization!"
    fi
    info "Organization created successfully!"
}

# Create a robot account in organization with the name informed via environment,
# using the super-user's ACCESS_TOKEN to authorize the request.
quay_create_robot_account() {
    local quay_url="https://${QUAY_HOSTNAME}/api/v1/organization/${QUAY_ORGANIZATION}/robots/$1"
    local data=(
        "{"
        "\"description\": \"Quay robot account for ${QUAY_ORGANIZATION}\","
        "\"unstructured_metadata\": {}"
        "}"
    )
    local create_response token

    info "Creating Quay robot account $1..."
    create_response=$(
        curl \
            --silent \
            --insecure \
            --location \
            --request PUT \
            --header 'Content-Type: application/json' \
            --header "Authorization: Bearer ${ACCESS_TOKEN}" \
            --data "${data[*]}" \
            "${quay_url}"
    )

    # When robot account already exists, the script should continue without failing.
    if [[ -z "${create_response}" || "${create_response}" == *"Existing robot"* ]]; then
        warn "Robot account already exists!"
        return 0
    fi

    # When robot account creation fails, the script should fail completely.
    if [[ -z "${create_response}" || ("${create_response}" != *"created"*) ]]; then
        fail "Failed to create robot account!"
    fi

    info "Extracting token from the response..."
    # When response doesn't contain the expected "token", the script should
    # fail completely.
    token=$(echo "${create_response}" | jq --raw-output '.token')
    if [[ -z "${token}" || "${token}" == "null" ]]; then
        fail "Failed to get robot account token!"
    fi

    info "Robot account $1 created successfully!"
    if [[ "$1" == "tssc_rw" ]]; then
        export QUAY_ROBOT_TOKEN="${token}"
        export QUAY_ROBOT_USERNAME="${QUAY_ORGANIZATION}+${QUAY_ROBOT_SHORT_NAME}"
    else
        export QUAY_ROBOT_TOKEN_READONLY="${token}"
        export QUAY_ROBOT_USERNAME_READONLY="${QUAY_ORGANIZATION}+${QUAY_ROBOT_SHORT_NAME_READONLY}"
    fi
}

# Create a new permission prototype in organization, that will automatically
# grant related permission of repositories to robot account
quay_create_permission_prototype() {
    local quay_url="https://${QUAY_HOSTNAME}/api/v1/organization/${QUAY_ORGANIZATION}/prototypes"
    local role
    if [[ "$1" == *"tssc_rw" ]]; then
        role="admin"
    else
        role="read"
    fi
    local data=(
        "{"
        "\"role\": \"${role}\","
        "\"activating_user\": {"
            "\"name\": \"\""
            "},"
        "\"delegate\": {"
            "\"name\": \"$1\","
            "\"kind\": \"user\""
            "}"
        "}"
    )
    local create_response

    info "Creating new permission prototype in organization ${QUAY_ORGANIZATION}..."
    create_response=$(
        curl \
            --silent \
            --insecure \
            --location \
            --request POST \
            --header 'Content-Type: application/json' \
            --header "Authorization: Bearer ${ACCESS_TOKEN}" \
            --data "${data[*]}" \
            "${quay_url}"
    )

    if [[ -z "${create_response}" || "${create_response}" != *"$1"* ]]; then
        fail "Failed to create new permission prototype!"
    fi

    info "Create new permission prototype successfully!"
}

# Create a new team in organization with creator role
quay_create_team() {
    local team_name="${QUAY_ORGANIZATION}-creator"
    local quay_url="https://${QUAY_HOSTNAME}/api/v1/organization/${QUAY_ORGANIZATION}/team/${team_name}"
    local data=(
        "{"
        "\"role\": \"creator\","
        "\"description\": \"Team with creator role for ${QUAY_ORGANIZATION}\""
        "}"
    )
    local create_response

    info "Creating new team with creator role in organization ${QUAY_ORGANIZATION}..."
    create_response=$(
        curl \
            --silent \
            --insecure \
            --location \
            --request PUT \
            --header 'Accept: application/json' \
            --header 'Content-Type: application/json' \
            --header "Authorization: Bearer ${ACCESS_TOKEN}" \
            --data "${data[*]}" \
            "${quay_url}"
    )

    if [[ -z "${create_response}" || "${create_response}" != *"${team_name}"* ]]; then
        fail "Failed to create new team with creator role!"
    fi

    info "Create new team with creator role successfully!"
}

## Assign the robot account to the team with creator role
quay_assign_robot_to_team() {
    local team_name="${QUAY_ORGANIZATION}-creator"
    local quay_url="https://${QUAY_HOSTNAME}/api/v1/organization/${QUAY_ORGANIZATION}/team/${team_name}/members/${QUAY_ROBOT_USERNAME}"

    local create_response

    info "Assigning robot account to team ${team_name}..."
    create_response=$(
        curl \
            --silent \
            --insecure \
            --location \
            --request PUT \
            --header "Authorization: Bearer ${ACCESS_TOKEN}" \
            "${quay_url}"
    )

    if [[ -z "${create_response}" || "${create_response}" != *"${QUAY_ROBOT_USERNAME}"* ]]; then
        fail "Failed to assign robot account to team!"
    fi

    info "Assign robot account to team successfully!"
}

# Initializes the Quay super-user and creates a "docker-registry" secret with the
# credentials informed via environment variables.
quay_helper() {
    info "Initializing Quay helper..."
    info "Quay hostname: '${QUAY_HOSTNAME}'"
    info "Quay username: '${QUAY_USERNAME}'"
    info "Secret namespace/name: '${NAMESPACE}/${SECRET_NAME}'"

    quay_initialize_super_user || {
        warn "Failed to initialize super-user!"
        return 1
    }

    # Checking if Quay has already been initialized, thus the secret doesn't need
    # to get recreated, the acccess token is only obtained during initialization
    if oc get secret "${SECRET_NAME}" --namespace="${NAMESPACE}" &>/dev/null &&
        [[ -z "${ACCESS_TOKEN}" ]]; then
        warn "Secret already exists, Quay has already been initialized!"
        return 0
    fi

    quay_create_organization

    for robot in "${QUAY_ROBOT_SHORT_NAME}" "${QUAY_ROBOT_SHORT_NAME_READONLY}"; do
        quay_create_robot_account "${robot}"
    done

    for robot in "$QUAY_ROBOT_USERNAME" "$QUAY_ROBOT_USERNAME_READONLY"; do
        quay_create_permission_prototype "${robot}"
    done

    quay_create_team
    quay_assign_robot_to_team

    return 0
}

#
# Main
#

retry_quay_helper() {
    assert_variables
    for i in {1..30}; do
        wait=$((i * 5))
        info "[${i}/30] Waiting for ${wait} seconds before retrying..."
        sleep ${wait}

        info "Trying to initialize Quay super-user..."  
        #Using subshell to contain the failure
        if ( quay_helper ); then
            return 0
        fi

        info "Attempt ${i} failed, retrying..."
    done
    return 1
}

if retry_quay_helper; then
    echo "# [INFO] Quay helper succeeded."
else
    fail "Quay helper failed!"
fi
