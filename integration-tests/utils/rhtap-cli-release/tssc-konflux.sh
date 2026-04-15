#!/usr/bin/env bash

# Script to create MR to create new application version and release in konflux
# Script updates the files and creates MR. It is left to
#   the user to get MR approved, merged and verify actions complete
#   successfully in konflux.
#   -k can be used to create a MR that will remove versions over the number to keep.
# NOTE: tssc-cli-stream.yaml is expected to be in order by version. Oldest version first entry,
#       new version last entry in file.

set -o errexit
set -o nounset
set -o pipefail

# Defaults
APP="tssc-cli"
VERSION=""
ALT_VERSION=""
KEEP_VERSIONS=""
REPOSITORY="${REPOSITORY:-konflux-release-data}"
GITLAB_ORG="${GITLAB_ORG:-releng}"
POSITIONAL_ARGS=()
KONFLUX_URL="https://konflux-ui.apps.stone-prd-rh01.pg1f.p1.openshiftapps.com"
KONFLUX_NAMESPACE="rhtap-shared-team-tenant"

# Files
STREAM_FILE="tenants-config/cluster/stone-prd-rh01/tenants/rhtap-shared-team-tenant/tssc-cli/tssc-cli-stream.yaml"
RP_FILE_DIR="tenants-config/cluster/stone-prd-rh01/tenants/rhtap-shared-team-tenant/tssc-cli"
RPA_FILE_DIR="config/stone-prd-rh01.pg1f.p1/product/ReleasePlanAdmission/rhtap-shared-team"

# Messages
NEXT_STEPS="\nMerge Request Creation - SUCCESSFUL, See above for MR's URL\n
\nNEXT STEPS:\n
	1. Verify/Get MR Approved\n
        2. Merge MR and Verify Successful\n
        3. Konflux: ($KONFLUX_URL) Namespace: ($KONFLUX_NAMESPACE) - Verify Application added in Konflux\n
           Release pipeline started. Once pipeline completes, verify creation of Release was Successful\n"
RELEASE_DOC="For detailed information on 'Release Steps' See: https://docs.google.com/document/d/1fxd-sq3IxLHWWqJM7Evhh9QeSXpqPMfRHHDBzAmT8-k/edit?tab=t.0#heading=h.9aaha887zz8f"

usage() {
    echo "
Usage:
    ${0##*/} [options] <version>
       <version> = Application version to create and release on konflux.

Optional arguments:
    --dry-run
        Do not push updates and create MR.
    -d, --debug
        Activate tracing/debug mode.
    -h, --help
        Display this message.
    -k, --keep
        Number of Versions to keep. Will keep this many versions and
          delete the others. NOTE: <version> should be omitted if not creating
          new application and release version.
    -w, --wip
        Set work in progress, MR will be set as Draft
Example:
    ${0##*/} 1.7
" >&2
}


parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
        --dry-run)
            DRY_RUN=1
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
        -k | --keep)
            if [[ -n "$2" && "$2" != -* ]]; then
                KEEP_VERSIONS="$2"
                shift
            else
                echo "Error: Option $1 requires an argument."
                echo ""
                usage
                exit 1
            fi
            ;;
        -w | --wip)
            WIP=1
            ;;
        --) # End of options
            break
            ;;
        -*) # Unknown option
            echo "Error: Unknown option $1"
            usage
            exit 1
            ;;
        *)
            POSITIONAL_ARGS+=("$1")
            if [[ ${#POSITIONAL_ARGS[@]} -gt 1 ]]; then
                echo "[ERROR] Unknown argument: $1"
                usage
                exit 1
            else
                VERSION="$1"
            fi
            ;;
        esac
        shift
    done

    if [[ -z "${KEEP_VERSIONS}" ]]; then
        if [[ "${#POSITIONAL_ARGS[@]}" -ne 1 ]]; then
            echo "[ERROR] Positional argument 'version' is required"
            usage
            exit 1
        fi
    fi
}


init() {
    TMP_DIR=$(mktemp -d)
    PROJECT_DIR="$TMP_DIR/$REPOSITORY"
    trap cleanup EXIT

    git clone git@gitlab.cee.redhat.com:$GITLAB_ORG/$REPOSITORY.git --branch main --single-branch "$PROJECT_DIR"
    cd "$PROJECT_DIR"
}


cleanup() {
    if [ -z "${DRY_RUN:-}" ]; then
        rm -rf "$TMP_DIR"
    else
        echo -e "\nYou can browse the repository: $PROJECT_DIR"
    fi
}


create_branch() {
    if [[ -z "${VERSION}" ]]; then
        BRANCH="${APP}-remove-unsupported-release"
    else
        BRANCH="${APP}-${VERSION}-add-release"
    fi
    git checkout -b "$BRANCH"
}


update_stream() {
    LINE="value: \"$VERSION\""
    if grep -q -x "^[[:space:]]*$LINE" "$STREAM_FILE"; then
        echo "Error: Version already exists."
        exit 1
    fi
    echo "---
apiVersion: projctl.konflux.dev/v1beta1
kind: ProjectDevelopmentStream
metadata:
  name: tssc-cli-release-${ALT_VERSION}
spec:
  project: tssc-cli
  template:
    name: tssc-cli
    values:
      - name: version
        value: \"$VERSION\"
      - name: branchName
        value: \"release-$VERSION\"" >> $STREAM_FILE
}

delete_old_vers() {
    NEW_NUM_VERSIONS=`yq eval-all '[.] | length' $STREAM_FILE`

    while  [[ $NEW_NUM_VERSIONS -gt $KEEP_VERSIONS ]] 
    do
        WORKING_VERSION=`yq eval-all 'select(documentIndex == 0) | .spec.template.values[] | select(.name == "version") | .value' $STREAM_FILE`
        WORKING_ALT_VERSION=$(echo "$WORKING_VERSION" | sed -r 's/\./-/g')

        # Delete old stream
        yq -i 'del(select(documentIndex == 0))' $STREAM_FILE

        # Delete old corresponding RP
        rm $RP_FILE_DIR/tssc-cli-rp-$WORKING_ALT_VERSION.yaml

        # Remove RP from kustomization.yaml
        sed -i "/tssc-cli-rp-$WORKING_ALT_VERSION.yaml/d" $RP_FILE_DIR/kustomization.yaml

        # Delete old corresponding RPA
        rm $RPA_FILE_DIR/tssc-cli-$WORKING_ALT_VERSION-prod.yaml

        NEW_NUM_VERSIONS=`yq eval-all '[.] | length' $STREAM_FILE`
    done
}

update_rp() {
    cp --update=none-fail $RP_FILE_DIR/tssc-cli-rp-$CURRENT_ALT_VERSION.yaml $RP_FILE_DIR/tssc-cli-rp-$ALT_VERSION.yaml
    SRCH_VERSION=$(echo "$CURRENT_VERSION" | sed -r 's/\./\\\./g')
    RPL_VERSION=$(echo "$VERSION" | sed -r 's/\./\\\./g')
    sed -i "s/$SRCH_VERSION/$RPL_VERSION/g" $RP_FILE_DIR/tssc-cli-rp-$ALT_VERSION.yaml
    sed -i "s/$CURRENT_ALT_VERSION/$ALT_VERSION/g" $RP_FILE_DIR/tssc-cli-rp-$ALT_VERSION.yaml
    echo "  - tssc-cli-rp-$ALT_VERSION.yaml" >> $RP_FILE_DIR/kustomization.yaml
}


run_build_manifests() {
    # Complete modifications by running build-manifest.sh
    echo -e "\nRunning build-manifests.sh"
    #./tenants-config/build-single.sh $KONFLUX_NAMESPACE > /dev/null
    ./tenants-config/build-manifests.sh > /dev/null

    if [ $? -ne 0 ]; then
        echo "Error: Running of build-manifests failed"
        exit 1
    fi

    echo -e "Running of build-manifests.sh - SUCCESSFUL\n"

    # Pause for 10
    sleep 10
}


update_rpa() {
    cp --update=none-fail $RPA_FILE_DIR/tssc-cli-$PREV_ALT_VERSION-prod.yaml $RPA_FILE_DIR/tssc-cli-$ALT_VERSION-prod.yaml
    SRCH_VERSION=$(echo "$PREV_VERSION" | sed -r 's/\./\\\./g')
    RPL_VERSION=$(echo "$VERSION" | sed -r 's/\./\\\./g')
    sed -i "s/$SRCH_VERSION/$RPL_VERSION/g" $RPA_FILE_DIR/tssc-cli-$ALT_VERSION-prod.yaml
    sed -i "s/$PREV_ALT_VERSION/$ALT_VERSION/g" $RPA_FILE_DIR/tssc-cli-$ALT_VERSION-prod.yaml
}


commit_code() {
    MESSAGE="release tssc-cli $VERSION.0 to quay.io/redhat-tssc/cli (Automated)"
    git add --all .
    git commit -m "$MESSAGE"
}


release() {
    NUM_VERSIONS=`yq eval-all '[.] | length' $STREAM_FILE`
    CURRENT_IDX=$((NUM_VERSIONS-1))
    VER_QUERY="yq eval-all 'select(documentIndex == $CURRENT_IDX) | .spec.template.values[] | select(.name == \"version\") | .value' $STREAM_FILE"
    CURRENT_VERSION=`eval "$VER_QUERY"`
    CURRENT_ALT_VERSION=$(echo "$CURRENT_VERSION" | sed -r 's/\./-/g')

    if [[ "$CURRENT_VERSION" != "$VERSION" ]]; then
        echo "[ERROR] Unable to release. App version $VERSION does not look to be the latest."
        exit 1
    fi

    PREV_IDX="0"
    VER_QUERY="yq eval-all 'select(documentIndex == $PREV_IDX) | .spec.template.values[] | select(.name == \"version\") | .value' $STREAM_FILE"
    PREV_VERSION=`eval "$VER_QUERY"`
    PREV_ALT_VERSION=$(echo "$PREV_VERSION" | sed -r 's/\./-/g')

    echo -e "Creating RPA file"
    update_rpa
    echo -e "Creation of RPA - SUCCESSFUL"
}


app() {
    NUM_VERSIONS=`yq eval-all '[.] | length' $STREAM_FILE`
    CURRENT_IDX=$((NUM_VERSIONS-1))
    VER_QUERY="yq eval-all 'select(documentIndex == $CURRENT_IDX) | .spec.template.values[] | select(.name == \"version\") | .value' $STREAM_FILE"
    CURRENT_VERSION=`eval "$VER_QUERY"`
    CURRENT_ALT_VERSION=$(echo "$CURRENT_VERSION" | sed -r 's/\./-/g')

    echo -e "\nUpdating application files"
    update_stream
    update_rp
    echo -e "Updating files - SUCCESSFUL\n"
}

push_changes() {
    DESCRIPTION="<h3>What:</h3>release tssc-cli $VERSION application to quay.io/redhat-tssc/cli<br /><h3>Why:</h3><br />"

    echo -e "\nPushing changes and creating MR\n"

    if [ -z "${WIP:-}" ]; then
        ADD_OPT=""
    else
        ADD_OPT="-o merge_request.draft"
    fi

    CREATE_MR_CMD="git push origin ${BRANCH} -o merge_request.create $ADD_OPT -o merge_request.target=main -o merge_request.description=\"$DESCRIPTION\" -o merge_request.remove_source_branch -o merge_request.squash=true -o merge_request.merge_when_pipeline_succeeds"

    eval "$CREATE_MR_CMD"

    echo -e $NEXT_STEPS
    echo -e "$RELEASE_DOC"
}

action() {
    init

    # Set alternate version #-#
    ALT_VERSION=$(echo "$VERSION" | sed -r 's/\./-/g')

    # Create branch to perform work
    create_branch

    if [[ -n "${VERSION}" ]]; then
        # Update application files
        app

        # Add Release Plan Admission
        release
    fi

    # Delete old unsupported versions.
    if [[ -n "${KEEP_VERSIONS}" ]]; then
        delete_old_vers
    fi

    # Run build_manifest
    run_build_manifests

    # Commit changes
    commit_code

    # Push changes if not a dry run
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
