#!/usr/bin/env bash

# Reset a service, namespace or cluster, which is helpful for
# development and testing.

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR="$(
    cd "$(dirname "$0")" >/dev/null
    pwd
)"

if [ "$(uname -s)" == "Darwin" ]; then
    READLINK="greadlink"
else
    READLINK="readlink"
fi

usage() {
    echo "
Usage:
    ${0##*/} [options]

Optional arguments:
    -a, --all
        Reset the cluster and integration services.
    -c, --cluster
        Reset all TSSC namespaces in the cluster.
    -e, --env-file
        Environment variables definitions (default: $SCRIPT_DIR/private.env)
    -n, --namespace NAMESPACE
        Delete the specified namespace. The option can be repeated
        to delete multiple namespaces.
    -i, --integration INTEGRATION
        Service to reset [bitbucket, github, gitlab]. The option can be repeated
        to reset multiple services.
    -d, --debug
        Activate tracing/debug mode.
    -h, --help
        Display this message.

Example:
    ${0##*/} -e private.env -i github -i gitlab
" >&2
}

parse_args() {
    ENVFILE="$SCRIPT_DIR/private.env"
    NAMESPACES=()
    while [[ $# -gt 0 ]]; do
        case $1 in
        -a|--all)
            CLUSTER=1
            BITBUCKET=1
            GITHUB=1
            GITLAB=1
            ;;
        -c|--cluster)
            CLUSTER=1
            ;;
        -e | --env-file)
            ENVFILE="$($READLINK -e "$2")"
            shift
            ;;
        -i|--integration)
            case $2 in
            bitbucket)
                BITBUCKET=1
                ;;
            github)
                GITHUB=1
                ;;
            gitlab)
                GITLAB=1
                ;;
            *)
                echo "[ERROR] Unknown integration: $1"
                usage
                ;;
            esac
            shift
            ;;
        -n|--namespace)
            NAMESPACES+=("$2")
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

    # shellcheck disable=SC1090
    source "$ENVFILE"
}

init() {
    PROJECT_DIR="$(
        cd "$(dirname "$SCRIPT_DIR")" >/dev/null
        pwd
    )"
    cd "$PROJECT_DIR"
}

action() {
    CONFIG="$PROJECT_DIR/installer/config.yaml"
    NAMESPACE="$(yq '.tssc.namespace' "$CONFIG")"

    if [ -n "${CLUSTER:-}" ]; then
        echo '# Cluster'
        for ns in $(kubectl get namespaces -o name | grep "/$NAMESPACE" | cut -d/ -f2); do
            NAMESPACES+=("$ns")
        done
        echo
    fi
    if [ -n "${NAMESPACES[*]}" ]; then
        echo "# Namespaces"
        for ns in "${NAMESPACES[@]}"; do
            kubectl delete namespace "$ns" &
            echo "Deleting namespace $ns..."
            for cr in "applications.argoproj.io" "persistentvolumeclaims"; do
                sleep 3
                while [ "$(oc get "$cr" -n "$ns" -o name | wc -l)" != "0" ]; do
                    for kt in $(oc get "$cr" -n "$ns" -o name); do
                        kubectl patch "$kt" -n "$ns" -p '{"metadata":{"finalizers": null}}' --type=merge
                    done
                done
            done
        done
        wait
        echo "✓ Deleted all namespaces"
        echo
    fi
    if [ -n "${BITBUCKET:-}" ]; then
        echo '# Bitbucket'
        CURL=(
            "curl"
            "--fail"
            "--header" "Authorization: Basic $(echo -n "$BITBUCKET__USERNAME:$BITBUCKET__APP_PASSWORD" | base64)" \
            "--header" "Content-Type: application/json"
            "--silent"
        )
        URL="https://api.$BITBUCKET__HOST/2.0"

        mapfile -t REPO_LIST < <( "${CURL[@]}" "$URL/repositories/$BITBUCKET__WORKSPACE" | yq ".values[] | .slug" )
        for REPO in "${REPO_LIST[@]}"; do
            "${CURL[@]}" --request DELETE "$URL/repositories/$BITBUCKET__WORKSPACE/$REPO"
            echo "✓ Deleted repository $REPO"
        done
        echo
    fi
    if [ -n "${GITHUB:-}" ]; then
        echo '# GitHub'
        for REPO in $(gh repo list "$GITHUB__ORG" --json url | yq '.[].url'); do
            gh repo delete --yes "$REPO"
        done
        echo
    fi
    if [ -n "${GITLAB:-}" ]; then
        echo '# GitLab'
        for REPO in $(glab repo list -g "$GITLAB__GROUP" --output json | yq '.[].path_with_namespace' | grep -v -E "deletion_scheduled-[0-9]*$"); do
            glab repo delete --yes "$REPO"
            echo "✓ Deleted repository $REPO"
        done 2>/dev/null
        echo
    fi
}

main() {
    parse_args "$@"
    init
    action
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
fi
