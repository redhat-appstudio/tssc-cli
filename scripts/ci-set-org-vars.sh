#!/usr/bin/env bash

# Set the variables/secrets in the CIs

# Script will require jq, az (azure client with azure-devops extention), oc and gh (github client)

set -o errexit
set -o nounset
set -o pipefail

usage() {
    echo "
Usage:
    ${0##*/} [options]

Mandatory arguments:
    -b, --backend CI_BACKEND
        Sets ci backend to one of this: github, gitlab, azure, jenkins (required)

Optional arguments:
    --dry-run
        Print the variables instead of pushing them to the CI
    -g, --group CI_GROUP
        The var group that stores the variables, ignored for github and gitlab, but will be used for azure (default: tssc)
    -n, --namespace NAMESPACE
        TSSC installation namespace (default: tssc)
    -p, --project CI_PROJECT
        The project that stores the variables, ignored for github and gitlab but required for azure
    -s, --source CI_SOURCE_CONTROL
        Sets ci source control to one of this: github, gitlab (default: github)
    -d, --debug
        Activate tracing/debug mode.
    -h, --help
        Display this message.

Example:
    ${0##*/} -b github
" >&2
}

parse_args() {
    NAMESPACE="tssc"
    CI_GROUP="tssc"
    CI_SOURCE_CONTROL="github"
    while [[ $# -gt 0 ]]; do
        case $1 in
        --dry-run)
            DRY_RUN="1"
            export DRY_RUN
            ;;
        -b | --backend)
            CI_BACKEND="$2"
            shift
            ;;
        -g | --group)
            CI_GROUP="$2"
            shift
            ;;
        -n | --namespace)
            NAMESPACE="$2"
            shift
            ;;
        -p | --project)
            CI_PROJECT="$2"
            shift
            ;;
        -s | --source)
            CI_SOURCE_CONTROL="$2"
            shift
            ;;
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

getValues() {
    COSIGN_SECRET_JSON=$(oc get secrets -n openshift-pipelines signing-secrets -o json)
    COSIGN_SECRET_KEY="$(echo "$COSIGN_SECRET_JSON" | jq -r '.data."cosign.key"')"
    COSIGN_SECRET_PASSWORD="$(echo "$COSIGN_SECRET_JSON" | jq -r '.data."cosign.password"')"
    COSIGN_PUBLIC_KEY="$(echo "$COSIGN_SECRET_JSON" | jq -r '.data."cosign.pub"')"

    TPA_SECRET="tssc-trustification-integration"
    TPA_SECRET_JSON=$(oc get secret -n "$NAMESPACE" "$TPA_SECRET" -o json)
    TPA_URL="$(echo "$TPA_SECRET_JSON" | jq -r '.data.bombastic_api_url | @base64d')"
    TPA_OIDC_ISSUER_URL="$(echo "$TPA_SECRET_JSON" | jq -r '.data.oidc_issuer_url | @base64d')"
    TPA_OIDC_CLIENT_ID="$(echo "$TPA_SECRET_JSON" | jq -r '.data.oidc_client_id | @base64d')"
    TPA_OIDC_CLIENT_SECRET="$(echo "$TPA_SECRET_JSON" | jq -r '.data.oidc_client_secret | @base64d')"
    TPA_SUPPORTED_CYCLONEDX_VERSION="$(echo "$TPA_SECRET_JSON" | jq -r '.data.supported_cyclonedx_version | @base64d')"

    TAS_SECRET="tssc-tas-integration"
    TAS_SECRET_JSON=$(oc get secret -n "$NAMESPACE" "$TAS_SECRET" -o json)
    TAS_REKOR_HOST="$(echo "$TAS_SECRET_JSON" | jq -r '.data.rekor_url | @base64d')"
    TAS_TUF_MIRROR="$(echo "$TAS_SECRET_JSON" | jq -r '.data.tuf_url | @base64d')"

    SECRET_VARS=("COSIGN_SECRET_KEY" "COSIGN_SECRET_PASSWORD" "GITOPS_AUTH_PASSWORD" "TRUSTIFICATION_OIDC_CLIENT_SECRET")
}

is_in_list() {
    local target="$1"
    shift
    for item in "$@"; do
        [[ "$item" == "$target" ]] && return 0
    done
    return 1
}

validateSourceControl() {
    if ! is_in_list "$CI_SOURCE_CONTROL" gitlab github; then
        echo "CI source control $CI_SOURCE_CONTROL is not supported"
        exit 1
    fi

    SECRET="tssc-$CI_SOURCE_CONTROL-integration"
    if oc get secrets -n "$NAMESPACE" "$SECRET" >/dev/null 2>&1 ; then
        echo "There is an integration secret for CI source control $CI_SOURCE_CONTROL"
    else
        echo "$SECRET required when selecting CI source control $CI_SOURCE_CONTROL"
        exit 1
    fi

}

validateBackend() {
    if [ -z "${CI_BACKEND:-}" ]; then
        echo "argument --backend (-b) is required"
        exit 1
    elif [[ "azure" == "$CI_BACKEND" && -z "${CI_PROJECT:-}" ]]; then
        echo "argument --project (-p) is required when backend is set to azure"
        exit 1
    fi

    if ! is_in_list "$CI_BACKEND" gitlab github azure jenkins; then
        echo "CI backend $CI_BACKEND is not supported"
        exit 1
    fi

    SECRET="tssc-$CI_BACKEND-integration"
    if oc get secrets -n "$NAMESPACE" "$SECRET" >/dev/null 2>&1 ; then
        echo "There is an integration secret for CI backend $CI_BACKEND"
    else
        echo "$SECRET required when selecting CI backend $CI_BACKEND"
        exit 1
    fi

}

setVars() {
    if [ -n "${GIT_ORG:-}" ]; then
        echo "Organization/Project: '$GIT_ORG'"
    fi
    setVar COSIGN_SECRET_PASSWORD "$COSIGN_SECRET_PASSWORD"
    setVar COSIGN_SECRET_KEY "$COSIGN_SECRET_KEY"
    setVar COSIGN_PUBLIC_KEY "$COSIGN_PUBLIC_KEY"
    setVar GITOPS_AUTH_PASSWORD "$GIT_TOKEN"
    setVar REKOR_HOST "$TAS_REKOR_HOST"
    setVar TRUSTIFICATION_BOMBASTIC_API_URL "$TPA_URL"
    setVar TRUSTIFICATION_OIDC_CLIENT_ID "$TPA_OIDC_CLIENT_ID"
    setVar TRUSTIFICATION_OIDC_CLIENT_SECRET "$TPA_OIDC_CLIENT_SECRET"
    setVar TRUSTIFICATION_OIDC_ISSUER_URL "$TPA_OIDC_ISSUER_URL"
    setVar TRUSTIFICATION_SUPPORTED_CYCLONEDX_VERSION "$TPA_SUPPORTED_CYCLONEDX_VERSION"
    setVar TUF_MIRROR "$TAS_TUF_MIRROR"
}

setVar() {
    NAME=$1
    VALUE=$2
    echo -n "Setting $NAME: "
    if [ -n "${DRY_RUN:-}" ]; then
        echo "$VALUE"
    else
        "${CI_BACKEND}SetVar"
        sleep .5 # rate limiting to prevent issues
    fi
}



githubGetValues() {
    SECRET="tssc-github-integration"
    SECRET_JSON=$(oc get secrets -n "$NAMESPACE" "$SECRET" -o json)
    GIT_ORG="$(echo "$SECRET_JSON" | jq -r '.data.ownerLogin | @base64d')"
    GIT_TOKEN="$(echo "$SECRET_JSON" | jq -r '.data.token | @base64d')"
    GIT_USER="$(echo "$SECRET_JSON" | jq -r '.data.username | @base64d')"
    export GH_TOKEN=$GIT_TOKEN
}

githubSetVar() {
    if is_in_list "$NAME" "${SECRET_VARS[@]}"; then
        gh secret set "$NAME" -b "$VALUE" --org "$GIT_ORG" --visibility all
    else
        gh variable set "$NAME" -b "$VALUE" --org "$GIT_ORG" --visibility all
    fi
    
}

azureGetValues() {
    SECRET="tssc-azure-integration"
    SECRET_JSON=$(oc get secrets -n "$NAMESPACE" "$SECRET" -o json)
    AZURE_ORG=$(echo "$SECRET_JSON" | jq -r '.data.organization | @base64d')
    AZURE_HOST=$(echo "$SECRET_JSON" | jq -r '.data.host | @base64d')
    AZURE_TOKEN=$(echo "$SECRET_JSON" | jq -r '.data.token | @base64d')
    AZURE_ORG_URL="https://$AZURE_HOST/$AZURE_ORG"
    AZURE_PROJECT=$CI_PROJECT
    AZURE_VAR_GROUP=$CI_GROUP
    AZURE_API_VERSION="7.1-preview.1"

    VAR_GROUP_JSON=$(curl -fsS -u ":$AZURE_TOKEN" \
      "$AZURE_ORG_URL/$AZURE_PROJECT/_apis/distributedtask/variablegroups?groupName=$AZURE_VAR_GROUP&api-version=$AZURE_API_VERSION")
    VAR_GROUP_ID=$(echo "$VAR_GROUP_JSON" | jq -r '.value[0].id')

    if [[ "$VAR_GROUP_ID" == "null" ]]; then
        VAR_GROUP_JSON=$(curl -fsS -X POST -u ":$AZURE_TOKEN" \
          "$AZURE_ORG_URL/$AZURE_PROJECT/_apis/distributedtask/variablegroups?api-version=$AZURE_API_VERSION" \
          -H "Content-Type: application/json" \
          -d @- \
<<EOF
        {
          "name": "$AZURE_VAR_GROUP",
          "variables": {
            "NAME": {
              "value": "$AZURE_VAR_GROUP"
            }
          },
          "type": "Vsts"
        }
EOF
        )

        VAR_GROUP_ID=$(echo "$VAR_GROUP_JSON" | jq -r '.id')
    else
        echo "Var Group $AZURE_VAR_GROUP already exists in project $AZURE_PROJECT"
        exit 1
    fi
}

azureUpdateVars() {
    if [[ -z "${CURRENT_VARS:-}" ]]; then
        CURRENT_VARS="$(
          jq -n \
            --arg group "$AZURE_VAR_GROUP" \
            --arg name "$NAME" \
            --arg value "$VALUE" \
            --arg isSecret "$IS_SECRET" \
            '{ name: $group, variables: { ($name): { value: $value, isSecret: $isSecret } } }'
        )"
    else
        CURRENT_VARS=$(echo "$CURRENT_VARS" | jq \
        --arg name "$NAME" \
        --arg value "$VALUE" \
        --arg isSecret "$IS_SECRET" \
        '.variables[$name] = { value: $value, isSecret: $isSecret }')
    fi
}

azureSetVar() {
    IS_SECRET=false
    if is_in_list "$NAME" "${SECRET_VARS[@]}"; then
        IS_SECRET=true
        echo "*********"
    else
        echo "$VALUE"
    fi


    azureUpdateVars

    echo "$CURRENT_VARS" | curl -fsS -X PUT "$AZURE_ORG_URL/$AZURE_PROJECT/_apis/distributedtask/variablegroups/$VAR_GROUP_ID?api-version=$AZURE_API_VERSION" \
    -H "Content-Type: application/json" \
    -u ":$AZURE_TOKEN" \
    -d @- \
    > /dev/null
}

gitlabGetValues() {
    SECRET="tssc-gitlab-integration"
    SECRET_JSON=$(oc get secrets -n "$NAMESPACE" "$SECRET" -o json)
    GIT_TOKEN="$(echo "$SECRET_JSON" | jq -r '.data.token | @base64d')"
    GIT_USER="$(echo "$SECRET_JSON" | jq -r '.data.username | @base64d')"
    GIT_ORG="$(echo "$SECRET_JSON" | jq -r '.data.group | @base64d')"
    URL="https://$(echo "$SECRET_JSON" | jq -r '.data.host | @base64d')"
    PID=$(curl -sL --proto "=https" --header "PRIVATE-TOKEN: $GIT_TOKEN" "$URL/api/v4/groups/$GIT_ORG" | jq -r ".id")
}

gitlabSetVar() {
  result=$(
    curl -s --request POST --header "PRIVATE-TOKEN: $GIT_TOKEN" \
      "$URL/api/v4/groups/$PID/variables" --form "key=$NAME" --form "value=$VALUE"
  )
  if echo "$result" | grep -q "has already been taken"; then
    result=$(
      curl -s --request PUT --header "PRIVATE-TOKEN: $GIT_TOKEN"  \
        "$URL/api/v4/groups/$PID/variables/$NAME" --form "value=$VALUE"
    )
  fi
  echo "$result" | jq --compact-output 'del(.description, .key, .value, .variable_type)'
}

# Functions to support Jenkins
add_secret() {
    local id=$1
    local secret=$2

    local json
    json=$(cat <<EOF
{
  "": "0",
  "credentials": {
    "scope": "GLOBAL",
    "id": "${id}",
    "secret": "${secret}",
    "description": "",
    "\$class": "org.jenkinsci.plugins.plaincredentials.impl.StringCredentialsImpl"
  }
}
EOF
)

    create_credentials "$json"    
}

add_username_with_password() {
    local id=$1
    local username=$2
    local password=$3

    local json
    json=$(cat <<EOF
{
  "": "0",
  "credentials": {
    "scope": "GLOBAL",
    "id": "${id}",
    "username": "${username}",
    "password": "${password}",
    "description": "",
    "\$class": "com.cloudbees.plugins.credentials.impl.UsernamePasswordCredentialsImpl"
  }
}
EOF
)

    create_credentials "$json"
}

create_credentials() {
    local json=$1

    curl -X POST "$JENKINS__URL/credentials/store/system/domain/_/createCredentials" \
    --user "$JENKINS__USERNAME:$JENKINS__TOKEN" \
    --data-urlencode "json=$json"
}

jenkinsGetValues() {
    SECRET="tssc-jenkins-integration"
    JENKINS_SECRET_JSON=$(oc get secrets -n "$NAMESPACE" "$SECRET" -o json)
    JENKINS__URL="$(echo "$JENKINS_SECRET_JSON" | jq -r '.data.baseUrl | @base64d')"
    JENKINS__USERNAME="$(echo "$JENKINS_SECRET_JSON" | jq -r '.data.username | @base64d')"
    JENKINS__TOKEN="$(echo "$JENKINS_SECRET_JSON" | jq -r '.data.token | @base64d')"

    # Add usernames with passwords
    if oc get secrets -n "$NAMESPACE" "tssc-quay-integration" -o name >/dev/null 2>&1; then
        echo "Setting QUAY_IO_CREDS: *********"
        add_username_with_password "QUAY_IO_CREDS" "$IMAGE_REGISTRY_USER" "$IMAGE_REGISTRY_PASSWORD"
    fi
    echo "Setting GITOPS_CREDENTIALS: *********"
    add_username_with_password "GITOPS_CREDENTIALS" "$GIT_USER" "$GIT_TOKEN"
}

jenkinsSetVar() {
    if is_in_list "$NAME" "${SECRET_VARS[@]}"; then
        echo "*********"
    else
        echo "$VALUE"
    fi
    add_secret "$NAME" "$VALUE"
}

main() {
    parse_args "$@"
    getValues
    validateSourceControl
    "${CI_SOURCE_CONTROL}GetValues"
    if [ -z "${DRY_RUN:-}" ]; then
        validateBackend
        echo "# $CI_BACKEND ##################################################"
        if [[ "$CI_SOURCE_CONTROL" != "$CI_BACKEND" ]]; then
            "${CI_BACKEND}GetValues"
        fi
    fi
    setVars
    echo
    echo "Success"
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
fi
