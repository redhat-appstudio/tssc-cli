#!/usr/bin/env bash

# Bumps the all the go direct dependencies one by one,
# ignoring versions that breaks the build.

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
    trap cleanup EXIT

    PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
    cd "$PROJECT_DIR"
}

cleanup() {
    rm -rf vendor/
    git restore .
}

update_dependency() {
    echo "# $DEPENDENCY"

    if ! go get -u "$DEPENDENCY"; then
		echo "[ERROR] \`go get -u $DEPENDENCY\` failed"
	    cleanup
	    return
	fi
    go mod verify
    if ! go mod tidy -v; then
		echo "[ERROR] \`go mod tidy\` failed"
	    cleanup
	    return
	fi

    if git diff --exit-code --quiet; then
        echo "No update"
        return
    fi

    go mod vendor
    if make; then
        git add .
        git commit -m "chore: bump go dependency $DEPENDENCY"
    else
		echo "[ERROR] \`make\` failed"
        cleanup
    fi
}

get_dependencies() {
    mapfile -t DEPENDENCIES < <(go list -mod=readonly -f '{{.Path}}' -m all)
}

action() {
    init
    get_dependencies
    for DEPENDENCY in "${DEPENDENCIES[@]}"; do
        if ! grep -qE "[[:space:]]${DEPENDENCY}[[:space:]]" go.mod; then
            continue
        fi
        echo
        update_dependency
        echo
    done
}

main() {
    parse_args "$@"
    action
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
fi
