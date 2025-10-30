#!/usr/bin/env bash
#
# Link a Secret to a ServiceAccount
#
shopt -s inherit_errexit
set -o errexit
set -o errtrace
set -o nounset
set -o pipefail

usage() {
    echo "
Usage:
    ${0##*/} --secret SECRET_NAME --serviceaccount SERVICEACCOUNT [options]

Mandatory arguments:
	--secret SECRET_NAME
		Name of the secret to link to the ServiceAccount (e.g. 'tssc-image-registry-auth')
    --serviceaccount SERVICEACCOUNT
        ServiceAccount to patch (e.g. 'pipeline')

Optional arguments:
    -d, --debug
        Activate tracing/debug mode.
    -h, --help
        Display this message.

Example:
    ${0##*/} --secret tssc-image-registry-auth --serviceaccount pipeline
" >&2
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
		--secret)
			SECRET_NAME="$2"
            shift
            ;;
        --serviceaccount)
            SERVICEACCOUNT="$2"
            shift
            ;;
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

    if [ -z "${SERVICEACCOUNT:-}" ]; then
        fail "Missing --serviceaccount argument."
    fi
    if [ -z "${SECRET_NAME:-}" ]; then
        fail "Missing --secret argument."
    fi
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

init() {
    TEMP_DIR="$(mktemp -d)"
    cd "$TEMP_DIR"
	trap cleanup EXIT

    SA_DEFINITION="service-account.yaml"
    SA_DEFINITION_UPDATED="$SA_DEFINITION.patch.yaml"
    [ -e "$SA_DEFINITION_UPDATED" ] && rm "$SA_DEFINITION_UPDATED"

    SECRET_NAME="tssc-image-registry-auth"
}

cleanup() {
    cd - >/dev/null
    rm -rf "$TEMP_DIR"
}

get_binaries() {
	command -v jq >/dev/null 2>&1 || fail "'jq' not found in PATH"

    if command -v kubectl >/dev/null 2>&1; then
        KUBECTL="kubectl"
        return
    fi
	if command -v oc >/dev/null 2>&1; then
        KUBECTL="oc"
        return
    fi

    fail "'kubectl' or 'oc' not found"
}

get_serviceaccount() {
	RETRIES=30
	for i in $(seq 1 $RETRIES); do
		if [ "$i" -gt 1 ]; then
			echo " Retrying in 10 seconds."
			sleep 10
		fi
		if "$KUBECTL" get serviceaccounts "$SERVICEACCOUNT" -o json >"$SA_DEFINITION" 2>/dev/null; then
			return 0
		fi
		echo -n "Failed to get ServiceAccount '$SERVICEACCOUNT' ($i/$RETRIES)."
	done
	echo
	fail "ServiceAccount '$SERVICEACCOUNT' not found"
}

get_secret() {
	RETRIES=30
	for i in $(seq 1 $RETRIES); do
		if [ "$i" -gt 1 ]; then
			echo " Retrying in 10 seconds."
			sleep 10
		fi
		if "$KUBECTL" get secret "$SECRET_NAME" >/dev/null 2>&1; then
			return 0
		fi
		echo -n "Failed to get Secret '$SECRET_NAME' ($i/$RETRIES)."
	done
	echo
	fail "Secret '$SECRET_NAME' not found"
}

patch_serviceaccount() {
    echo -n "Linking '$SECRET_NAME' to '$SERVICEACCOUNT': "

	jq --arg NAME "$SECRET_NAME" \
		'.secrets |= (. + [{"name": $NAME}] | unique)' \
		"$SA_DEFINITION" >"$SA_DEFINITION_UPDATED"
	cp "$SA_DEFINITION_UPDATED" "$SA_DEFINITION"
	if [ "$("$KUBECTL" get secret "$SECRET_NAME" -o jsonpath="{.type}")" = "kubernetes.io/dockerconfigjson" ]; then
		jq --arg NAME "$SECRET_NAME" \
		'.imagePullSecrets |= (. + [{"name": $NAME}] | unique)' \
		"$SA_DEFINITION" >"$SA_DEFINITION_UPDATED"
        cp "$SA_DEFINITION_UPDATED" "$SA_DEFINITION"
    fi
    OUTPUT=$("$KUBECTL" apply -f "$SA_DEFINITION_UPDATED" 2>&1) \
		|| { echo "Failed"; echo "$OUTPUT"; fail "Failed to patch ServiceAccount '$SERVICEACCOUNT'"; }
    echo "OK"
}

#
# Main
#
main() {
    parse_args "$@"
	init
    get_binaries
	get_secret
	get_serviceaccount
    patch_serviceaccount
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
    echo
	echo "Success"
fi
