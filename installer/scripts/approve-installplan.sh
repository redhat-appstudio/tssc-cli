#!/usr/bin/env bash
#
# Approve InstallPlans in the namespace.
#

shopt -s inherit_errexit
set -o errexit
set -o errtrace
set -o nounset
set -o pipefail

usage() {
    echo "
Usage:
    ${0##*/} --namespace NAMESPACE [options]

Mandatory arguments:
    --csvs CSVS
        Comma-separated list of CSVs to approve (e.g. 'operator1.v1.0.0,operator2.v2.1.1')
    -n, --namespace NAMESPACE
        TSSC installation namespace (default: tssc)

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
        case $1 in
        --csvs)
            CSVS="$2"
            shift
            ;;
        -n | --namespace)
            NAMESPACE="$2"
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
        esac
        shift
    done

    if [ -z "${NAMESPACE:-}" ]; then
        fail "Missing --namespace argument."
    fi
    if [ -z "${CSVS:-}" ]; then
        fail "Missing --csvs argument."
    fi
    IFS=',' read -ra CSVS <<< "$CSVS"
}

#
# Functions
#

fail() {
    echo "# [ERROR] ${*}" >&2
    exit 1
}

info() {
    echo "# [INFO] ${*}"
}

checkInstalledCSVs() {
	# Check if all required CSVs are already installed,
	# which signals that there's no need to approve an InstallPlan.
    echo "Checking installed ClusterServiceVersions"
    for CSV in "${CSVS[@]}"; do
        echo -n "  - $CSV: "
        if ! oc get clusterserviceversions "$CSV" -n "$NAMESPACE" >/dev/null 2>&1; then
            echo "Not found"
            echo
            return 1
        fi
        echo " Installed"
    done
    echo
}

findCSV() {
	# Check that a CSV is installed by the InstallPlan.
    for candidate in "${installCSVS[@]}"; do
        if [[ "$CSV" == "$candidate" ]]; then
            return
        fi
    done
    echo "No match found"
    fail "Unexpected version(s) in the InstallPlan"
}

approve_installplans() {
	# Approve InstallPlans that install the required CSVs

    # Source: https://github.com/redhat-cop/gitops-catalog/blob/main/installplan-approver/base/installplan-approver-job.yaml
    echo "Approving operator(s) install in '$NAMESPACE'.  Waiting a few seconds to make sure the InstallPlan gets created first."
    sleep 30
    echo

    for installplan in $(
        oc get installplans -n "$NAMESPACE" -o json \
          | jq -r '.items[] | select(.spec.approved == false) | .metadata.name'
    ); do
        echo "Checking InstallPlan '$installplan' clusterServiceVersionNames: "
        mapfile -t installCSVS < <(oc get installplans.operators.coreos.com -n "$NAMESPACE" "$installplan" -o jsonpath="{.spec.clusterServiceVersionNames}" | jq -r ".[]")
        for CSV in "${CSVS[@]}"; do
            echo -n "  - $CSV: "
            findCSV
            echo "OK"
        done

        echo -n "Approving InstallPlan '$installplan'"
        oc patch installplan "$installplan" -n "$NAMESPACE" --type=json -p='[{"op":"replace","path": "/spec/approved", "value": true}]'
        echo
    done
}

#
# Main
#
main() {
    parse_args "$@"
    if ! checkInstalledCSVs; then
        approve_installplans
    fi
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
    echo "Success"
fi
