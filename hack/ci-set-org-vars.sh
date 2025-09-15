#!/usr/bin/env bash

# Set the variables/secrets in the CIs

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
    -e, --env-file ENVFILE
        Environment variables definitions (default: $SCRIPT_DIR/private.env)
    -n, --namespace NAMESPACE
        TSSC installation namespace (default: tssc)
    --dry-run
        Print the variables instead of pushing them to the CI
    -d, --debug
        Activate tracing/debug mode.
    -h, --help
        Display this message.

Example:
    ${0##*/} -e private.env
" >&2
}

parse_args() {
    NAMESPACE="tssc"
    ENVFILE="$SCRIPT_DIR/private.env"
    while [[ $# -gt 0 ]]; do
        case $1 in
        -e | --env-file)
            ENVFILE="$(readlink -e "$2")"
            shift
            ;;
        -n | --namespace)
            NAMESPACE="$2"
            shift
            ;;
        --dry-run)
            DRY_RUN="1"
            export DRY_RUN
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

    # shellcheck disable=SC1090
    source "$ENVFILE"
}

getValues() {
    COSIGN_SECRET_KEY="$(oc get secrets -n openshift-pipelines signing-secrets -o json | yq '.data.["cosign.key"]')"
    COSIGN_SECRET_PASSWORD="$(oc get secrets -n openshift-pipelines signing-secrets -o json | yq '.data.["cosign.password"]')"
    COSIGN_PUBLIC_KEY="$(oc get secrets -n openshift-pipelines signing-secrets -o json | yq '.data.["cosign.pub"]')"

    for REGISTRY in "artifactory" "nexus" "quay"; do
        REGISTRY_SECRET="tssc-$REGISTRY-integration" # notsecret
        if ! oc get secrets -n "$NAMESPACE" "$REGISTRY_SECRET" -o name >/dev/null 2>&1; then
            continue
        fi
        REGISTRY_ENDPOINT="$(oc get secrets -n "$NAMESPACE" "$REGISTRY_SECRET" -o json | jq -r '.data.url | @base64d')"
        IMAGE_REGISTRY="${REGISTRY_ENDPOINT//https:\/\//}"
        IMAGE_REGISTRY_USER="$(
            oc get secrets -n "$NAMESPACE" "$REGISTRY_SECRET" -o json \
            | jq -r '.data.".dockerconfigjson" | @base64d' \
            | jq -r '.auths | to_entries[0].value.auth | @base64d' \
            | cut -d: -f1
        )"
        IMAGE_REGISTRY_PASSWORD="$(
            oc get secrets -n "$NAMESPACE" "$REGISTRY_SECRET" -o json \
            | jq -r '.data.".dockerconfigjson" | @base64d' \
            | jq -r '.auths | to_entries[0].value.auth | @base64d' \
            | cut -d: -f2-
        )"
        break
    done

    SECRET="tssc-acs-integration"
    ROX_CENTRAL_ENDPOINT="$(oc get secrets -n "$NAMESPACE" "$SECRET" -o json | yq '.data.endpoint | @base64d')"
    ROX_API_TOKEN="$(oc get secrets -n "$NAMESPACE" "$SECRET" -o json | yq '.data.token | @base64d')"

    REKOR_HOST="https://$(oc get routes -n tssc-tas -l "app.kubernetes.io/name=rekor-server" -o jsonpath="{.items[0].spec.host}")"
    TUF_MIRROR="https://$(oc get routes -n tssc-tas -l "app.kubernetes.io/name=tuf" -o jsonpath="{.items[0].spec.host}")"
}

getSCMs() {
    SCM_LIST=( )
    for SCM in github gitlab; do
        SECRET="tssc-$SCM-integration" # notsecret
        if kubectl get secrets -n "$NAMESPACE" "$SECRET" >/dev/null 2>&1 ; then
            SCM_LIST+=("$SCM")
        fi
    done
}

setVars() {
    if [ -n "${GIT_ORG:-}" ]; then
        echo "Organization/Project: '$GIT_ORG'"
    fi
    setVar COSIGN_SECRET_PASSWORD "$COSIGN_SECRET_PASSWORD"
    setVar COSIGN_SECRET_KEY "$COSIGN_SECRET_KEY"
    setVar COSIGN_PUBLIC_KEY "$COSIGN_PUBLIC_KEY"
    setVar GITOPS_AUTH_PASSWORD "$GIT_TOKEN"
    setVar IMAGE_REGISTRY "$IMAGE_REGISTRY"
    setVar IMAGE_REGISTRY_PASSWORD "$IMAGE_REGISTRY_PASSWORD"
    setVar IMAGE_REGISTRY_USER "$IMAGE_REGISTRY_USER"
    setVar QUAY_IO_CREDS_PSW "$IMAGE_REGISTRY_PASSWORD"
    setVar QUAY_IO_CREDS_USR "$IMAGE_REGISTRY_USER"
    setVar REKOR_HOST "$REKOR_HOST"
    setVar ROX_CENTRAL_ENDPOINT "$ROX_CENTRAL_ENDPOINT"
    setVar ROX_API_TOKEN "$ROX_API_TOKEN"
    setVar TUF_MIRROR "$TUF_MIRROR"
}

setVar() {
    NAME=$1
    VALUE=$2
    echo -n "Setting $NAME: "
    if [ -n "${DRY_RUN:-}" ]; then
        echo "'$VALUE'"
    else
        "${SCM}SetVar"
        sleep .5 # rate limiting to prevent issues
    fi
}

githubGetValues() {
    SECRET="tssc-github-integration"
    GIT_ORG="$GITHUB__ORG"
    GIT_TOKEN="$(oc get secrets -n "$NAMESPACE" "$SECRET" -o json | yq '.data.token | @base64d')"
}

githubSetVar() {
    gh secret set "$NAME" -b "$VALUE" --org "$GIT_ORG" --visibility all
}

gitlabGetValues() {
    SECRET="tssc-gitlab-integration"
    GIT_TOKEN="$(oc get secrets -n "$NAMESPACE" "$SECRET" -o json | yq '.data.token | @base64d')"
    GIT_ORG="$GITLAB__GROUP"
    URL="https://$(oc get secrets -n "$NAMESPACE" "$SECRET" -o json | yq '.data.host | @base64d')"
    PID=$(curl -sL --proto "=https" --header "PRIVATE-TOKEN: $GIT_TOKEN" "$URL/api/v4/groups/$GIT_ORG" | jq ".id")
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

main() {
    parse_args "$@"
    getValues
    getSCMs
    for SCM in "${SCM_LIST[@]}"; do
        echo "# $SCM ##################################################"
        "${SCM}GetValues"
        setVars
        echo
    done
    echo "Success"
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
fi
