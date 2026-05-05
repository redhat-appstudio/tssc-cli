#!/usr/bin/env bash
# Sync vendor/github.com/redhat-appstudio/helmet from a branch on your fork.
# The canonical module path stays github.com/redhat-appstudio/helmet; go.mod uses
# replace => your fork (go get github.com/dperaza4dustbit/helmet@… does not work
# because the module self-declares as redhat-appstudio/helmet).
#
# Usage (from repo root):
#   ./hack/vendor-helmet-fork.sh
#   HELMET_FORK=github.com/you/helmet HELMET_BRANCH=my-feature ./hack/vendor-helmet-fork.sh
#
# Alternative for day-to-day work with a local clone (no revisions at all):
#   go mod edit -replace=github.com/redhat-appstudio/helmet=../helmet
#   go mod tidy && go mod vendor

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

FORK="${HELMET_FORK:-github.com/dperaza4dustbit/helmet}"
BRANCH="${HELMET_BRANCH:-composable_bundles}"
DOMAIN="${HELMET_FORK_DOMAIN:-github.com/dperaza4dustbit}"

# Avoid sumdb/proxy quirks for private forks (no-op for public repos).
export GOPRIVATE="${GOPRIVATE:-${DOMAIN}/*}"

if ! command -v git >/dev/null 2>&1; then
	echo "git is required (for ls-remote)" >&2
	exit 1
fi

REMOTE="https://${FORK}.git"
TIP="$(git ls-remote "${REMOTE}" "refs/heads/${BRANCH}" | awk '{print $1}')"
if [[ -z "${TIP}" ]]; then
	echo "Could not resolve branch \"${BRANCH}\" on ${FORK} (${REMOTE})." >&2
	exit 1
fi

go mod edit -replace="github.com/redhat-appstudio/helmet=${FORK}@${TIP}"
go mod tidy
go mod vendor

echo "Helmet vendor synced: ${FORK}@${BRANCH} => ${TIP:0:12}… (full SHA in go.mod replace line)."
