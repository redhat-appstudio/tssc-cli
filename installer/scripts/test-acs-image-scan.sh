#!/usr/bin/env bash
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

get_roxctl() {
  info "Download roxctl cli from ${ROX_CENTRAL_ENDPOINT}"
  curl --fail --insecure -s -L --proto "=https" -H "Authorization: Bearer $ROX_API_TOKEN" \
    "https://${ROX_CENTRAL_ENDPOINT}/api/cli/download/roxctl-linux" \
    --output ./roxctl  \
    > /dev/null \
    || fail "Failed to download roxctl"
  chmod +x ./roxctl > /dev/null
}

test_scanner() {
  info "# Testing image scan"
  for i in $(seq 1 "${RETRIES}"); do
    wait=30
    echo
    date
    echo "### [${i}/${RETRIES}] roxctl image scan"
    if ./roxctl image scan \
        "--insecure-skip-tls-verify" \
        -e "${ROX_CENTRAL_ENDPOINT}" \
        --image "$IMAGE" \
        --output json \
        --force; then
      break
    fi
    if [ "$i" -eq "${RETRIES}" ]; then
      fail "Failed to test ACS scanner"
    fi
    echo "# Waiting for ${wait} seconds before retrying..."
    sleep ${wait}
  done
  info "# ACS scanner tested successfully"
}

#
# Main
#
main() {
  parse_args "$@"

  # Number of retries to attempt before giving up.
  declare -r RETRIES=${RETRIES:-90}

  get_roxctl
  test_scanner
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
  main "$@"
  echo
  echo "Success"
fi
