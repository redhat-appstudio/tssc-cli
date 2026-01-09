#!/usr/bin/env bash

# Manage a full deployment of TSSC based on values from an env file
# and a few parameters

# shellcheck disable=SC2016

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
    -c, --container NAME
        Run the cli from a container. Use 'dev' to build from source.
    -e, --env-file
        Environment variables definitions (default: $SCRIPT_DIR/private.env)
    -i, --integration INTEGRATION
        Use an external service [bitbucket, cert-manager, ci, github, gitlab,
        jenkins, tas].
    -d, --debug
        Activate tracing/debug mode.
    -h, --help
        Display this message.

Example:
    ${0##*/} -e private.env -i cert-manager
" >&2
}

parse_args() {
    PROJECT_DIR="$(
        cd "$(dirname "$SCRIPT_DIR")" >/dev/null
        pwd
    )"
    CI_VAR_DIR="$PROJECT_DIR/scripts"
    ENVFILE="$SCRIPT_DIR/private.env"
    KUBECONFIG="${KUBECONFIG:-$HOME/.kube/config}"
    CLI_BIN="run_bin"
    CLI="$CLI_BIN"
    while [[ $# -gt 0 ]]; do
        case $1 in
        -c|--container)
            CLI_IMAGE="$2"
            CLI="run_container"
            CLI_PORT="8228"
            shift
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
            cert-manager)
                CERT_MANAGER=1
                ;;
            ci)
                CI=1
                ;;
            github)
                GITHUB=1
                ;;
            gitlab)
                GITLAB=1
                ;;
            jenkins)
                JENKINS=1
                ;;
            tas)
                TAS=1
                ;;
            *)
                echo "[ERROR] Unknown integration: $1"
                usage
                ;;
            esac
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
    cd "$PROJECT_DIR"
}

build() {
    if [ "$CLI" = "run_bin" ]; then
        make
    else
        if [ "${CLI_IMAGE:-}" = "dev" ]; then
            make image-podman
            CLI_IMAGE="ghcr.io/redhat-appstudio/tssc:latest"
        fi
    fi
}

init_config() {
    CONFIG_DIR="$SCRIPT_DIR/config"
    mkdir -p "$CONFIG_DIR"
    CONFIG="$CONFIG_DIR/config.yaml"
    VALUES="$CONFIG_DIR/values.yaml.tpl"

    unshare

    cp "$PROJECT_DIR/installer/config.yaml" "$CONFIG"
    cp "$PROJECT_DIR/installer/charts/values.yaml.tpl" "$VALUES"
    cp "$KUBECONFIG" "$CONFIG_DIR/kubeconfig"
    KUBECONFIG="$CONFIG_DIR/kubeconfig"

    # shellcheck disable=SC1090
    source "$ENVFILE"
}

tssc_cli() {
    $CLI "$@"
}

run_bin() {
    eval "$PROJECT_DIR/bin/tssc $*"
}

run_container() {
    podman run \
        --entrypoint="bash" \
        --env-file="$ENVFILE" \
        --publish "$CLI_PORT:$CLI_PORT" \
        --rm \
        --volume="$KUBECONFIG:/tssc/.kube/config:Z,U" \
        --volume="$CONFIG:/tssc/$(basename "$CONFIG"):Z,U" \
        "$CLI_IMAGE" \
        -c "tssc $*"
    unshare
}

unshare() {
    if [ "$(uname -s)" != "Darwin" ]; then
        podman unshare chown -R 0:0 "$CONFIG_DIR"
    fi
}

configure() {
    if [[ -n "${CERT_MANAGER:-}" ]]; then
        yq -i '(.tssc.products[] | select(.name == "Cert-Manager")).enabled = false' "$CONFIG"
    fi
    if [[ -n "${CI:-}" ]]; then
        sed -i 's/\( *ci\): .*/\1: true/' "$VALUES"
    fi
    cd "$(dirname "$CONFIG")"
    tssc_cli config --force --get --create "$(basename "$CONFIG")"

    NAMESPACE="$(
        kubectl get configmap \
        -A \
        --selector "tssc.redhat-appstudio.github.com/config=true" \
        -o jsonpath="{.items[0].metadata.namespace}"
    )"
    export NAMESPACE

    cd - >/dev/null
}

integrations() {
    if [[ -n "${BITBUCKET:-}" ]]; then
        tssc_cli integration bitbucket --force \
            --app-password='"$BITBUCKET__APP_PASSWORD"' \
            --host='"$BITBUCKET__HOST"' \
            --username='"$BITBUCKET__USERNAME"'
    fi
    if [[ -n "${GITHUB:-}" ]]; then
        if ! kubectl get secret -n "$NAMESPACE" tssc-github-integration >/dev/null 2>&1; then
            tssc_cli integration github-app \
                --create \
                --token='"$GITHUB__ORG_TOKEN"' \
                --org='"$GITHUB__ORG"' \
                "tssc-$GITHUB__ORG-$(date +%m%d-%H%M)"
        fi
    fi
    if [[ -n "${GITLAB:-}" ]]; then
        if [[ -n "${GITLAB__APP__CLIENT__ID:-}" && -n "${GITLAB__APP__CLIENT__SECRET:-}" ]]; then
            tssc_cli integration gitlab --force \
                --app-id='"$GITLAB__APP__CLIENT__ID"' \
                --app-secret='"$GITLAB__APP__CLIENT__SECRET"' \
                --group='"$GITLAB__GROUP"' \
                --host='"$GITLAB__HOST"' \
                --token='"$GITLAB__TOKEN"'
        else
            tssc_cli integration gitlab --force \
                --group='"$GITLAB__GROUP"' \
                --host='"$GITLAB__HOST"' \
                --token='"$GITLAB__TOKEN"'
        fi
    fi
    if [[ -n "${JENKINS:-}" ]]; then
        tssc_cli integration jenkins --force \
            --token='"$JENKINS__TOKEN"' \
            --url='"$JENKINS__URL"' \
            --username='"$JENKINS__USERNAME"'
    fi
    if [[ -n "${QUAY:-}" ]]; then
        tssc_cli integration quay --force \
            --dockerconfigjson='"$QUAY__DOCKERCONFIGJSON"' \
            --token='"$QUAY__API_TOKEN"' --url='"$QUAY__URL"'
    fi
    if [[ -n "${TAS:-}" ]]; then
        tssc_cli integration trusted-artifact-signer --force \
            --rekor-url='"$TAS__REKOR_URL"' \
            --tuf-url='"$TAS__TUF_URL"'
    fi
}

deploy() {
    time tssc_cli deploy "${DEBUG:-}"
}

configure_ci() {
    if [[ -n "${GITHUB:-}" ]]; then
        "$CI_VAR_DIR/ci-set-org-vars.sh" --backend github
    fi
    if [[ -n "${GITLAB:-}" ]]; then
        "$CI_VAR_DIR/ci-set-org-vars.sh" --backend gitlab
    fi
}

action() {
    build
    init_config
    configure
    integrations
    deploy
    configure_ci
}

main() {
    parse_args "$@"
    init
    action
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
fi
