#!/usr/bin/env bash

# Creates a release branch, and sets tags.

set -o errexit
set -o nounset
set -o pipefail

usage() {
    echo "
Usage:
    ${0##*/} [options]

Optional arguments:
    --dry-run
        Do not push updates back to the upstream.
    -d, --debug
        Activate tracing/debug mode.
    -h, --help
        Display this message.

Example:
    ${0##*/}
" >&2
}

parse_args() {
    GIT_URL="git@github.com:redhat-appstudio/tssc-cli.git"
    while [[ $# -gt 0 ]]; do
        case $1 in
        --dry-run)
            DRY_RUN=1
            GIT_URL="https://github.com/redhat-appstudio/tssc-cli.git"
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

init() {
    TMP_DIR=$(mktemp -d)
    PROJECT_DIR="$TMP_DIR/tssc-cli"
    trap cleanup EXIT

    git clone "$GIT_URL" "$PROJECT_DIR"
    cd "$PROJECT_DIR"
}

cleanup() {
    if [ -z "${DRY_RUN:-}" ]; then
        rm -rf "$TMP_DIR"
    else
        echo "You can browse the repository: $PROJECT_DIR"
    fi
}

subscription_values_for_channel_lookup() {
    case "$1" in
    openshiftKeycloak) echo "installer/bundles/iam/charts/tssc-iam-subscriptions/values.yaml" ;;
    openshiftGitOps) echo "installer/bundles/gitops/charts/tssc-gitops-subscriptions/values.yaml" ;;
    openshiftPipelines) echo "installer/bundles/pipelines/charts/tssc-pipelines-subscriptions/values.yaml" ;;
    openshiftTrustedArtifactSigner) echo "installer/bundles/tas/charts/tssc-tas-subscriptions/values.yaml" ;;
    advancedClusterSecurity) echo "installer/bundles/acs/charts/tssc-acs-subscriptions/values.yaml" ;;
    developerHub) echo "installer/bundles/dh/charts/tssc-dh-subscriptions/values.yaml" ;;
    *)
        echo "[ERROR] Unknown subscription key for channel lookup: $1" >&2
        exit 1
        ;;
    esac
}

get_version() {
    local values_file
    values_file="$(subscription_values_for_channel_lookup "$1")"
    VERSION_XY="$(
        yq '.subscriptions.'"$1"'.channel' "$values_file" \
        | grep --extended-regexp "[0-9.]*" --only-matching
    )"
    VERSION_XYZ="$VERSION_XY.0"
    export VERSION_XY VERSION_XYZ
}

update_charts() {
    # Bump "version" in all charts
    get_version "developerHub"
    export RELEASE_BRANCH="release-$VERSION_XY"
    find installer/charts installer/bundles -name Chart.yaml | while read -r CHART; do
        yq -i '.version = strenv(VERSION_XYZ)' "$CHART"
    done

    # Bump "appVersion" in all charts
    get_version "openshiftGitOps"
    yq -i '.appVersion = strenv(VERSION_XY)' "installer/bundles/gitops/charts/tssc-gitops/Chart.yaml"
    get_version "openshiftPipelines"
    yq -i '.appVersion = strenv(VERSION_XY)' "installer/bundles/pipelines/charts/tssc-pipelines/Chart.yaml"
    get_version "openshiftTrustedArtifactSigner"
    yq -i '.appVersion = strenv(VERSION_XY)' "installer/bundles/tas/charts/tssc-tas/Chart.yaml"
    get_version "advancedClusterSecurity"
    yq -i '.appVersion = strenv(VERSION_XY)' "installer/bundles/acs/charts/tssc-acs/Chart.yaml"
    yq -i '.appVersion = strenv(VERSION_XY)' "installer/bundles/acs/charts/tssc-acs-test/Chart.yaml"
    get_version "developerHub"
    yq -i '.appVersion = strenv(VERSION_XY)' "installer/bundles/dh/charts/tssc-dh/Chart.yaml"
}

update_template() {
    CONFIG="installer/config.yaml"
    get_version "developerHub"
    CATALOG_URL="https://github.com/redhat-appstudio/tssc-dev-multi-ci/blob/release-v${VERSION_XY}.x/samples/all.yaml" \
    yq -i '(.tssc.products[] | select( .name == "Developer Hub") | .properties.catalogURL) = strenv(CATALOG_URL)' "$CONFIG"
}

commit_freeze() {
    git commit \
        --all \
        --message "chore: code freeze for $RELEASE_BRANCH"
    git tag "v$VERSION_XY-freeze"
}

update_ci() {
    for PLR in ".tekton/tssc-cli-pull-request.yaml" ".tekton/tssc-cli-push.yaml"; do
        sed -i --regexp-extended "s|== \"main\"|== \"$RELEASE_BRANCH\"|" "$PLR"
        sed -i --regexp-extended "s|  *appstudio\.openshift\.io/application: tssc-cli|\0-${VERSION_XY//./-}|" "$PLR"
        sed -i --regexp-extended "s|  *appstudio\.openshift\.io/component: tssc-cli|\0-${VERSION_XY//./-}|" "$PLR"
    done
    yq -i '.spec.params |= map(select(.name != "image-expires-after"))' ".tekton/tssc-cli-push.yaml"
    yq -i '(.spec.pipelineSpec.tasks[] | select(.name == "apply-tags") | .params[] | select(.name == "ADDITIONAL_TAGS") | .value[0]) = strenv(RELEASE_BRANCH)' ".tekton/tssc-cli-push.yaml"
}

commit_release() {
    git switch -c "$RELEASE_BRANCH"
    git commit \
        --all \
        --message "chore: update PLR to setup Konflux for $RELEASE_BRANCH"
}

code_freeze() {
    update_charts
    update_template
    commit_freeze
}

release_branch() {
    update_ci
    commit_release
}

push_changes() {
    git push --tags --set-upstream origin main
    git push --set-upstream origin "$RELEASE_BRANCH"
}

action() {
    init
    code_freeze
    release_branch
    if [ -z "${DRY_RUN:-}" ]; then
        push_changes
    fi
}

main() {
    parse_args "$@"
    action
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
fi
