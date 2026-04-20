#!/usr/bin/env bash
#
# Tear down TSSC Helm releases (and optionally application namespaces) on an OpenShift
# cluster so you can re-run a fresh install vs upgrade tests without provisioning a new cluster.
#
# Installs are distributed: Helmet/composable bundles emit multiple tssc-* Helm releases across
# installer and product namespaces (e.g. tssc-dh, tssc-keycloak). This script uses helm list -A
# and uninstalls every release whose name starts with tssc- — not a single umbrella chart.
#
# Requires: oc (logged in), helm, jq
#
# Usage:
#   ./hack/cluster-cleanup-tssc.sh
#   ./hack/cluster-cleanup-tssc.sh -d
#   ./hack/cluster-cleanup-tssc.sh -n my-installer-ns -a -o
#
# Does NOT delete openshift-* system namespaces (openshift-operators, openshift-pipelines, …).
# With -a/--aggressive it CAN delete operator hub namespaces created for this product (rhacs-operator,
# rhbk-operator, …); use only when you intend to remove those operators from the cluster.
#
# Closest to an unused cluster (recommended full reset): ./hack/cluster-cleanup-tssc.sh -a -o
# (-o removes Subscription CRs Helm keeps; -a removes rhacs/rhbk operator hub namespaces.)
#
# Subscription CRs use helm.sh/resource-policy: keep — Helm uninstall leaves them; use -o/--olm
# to delete the Operator Hub Subscriptions this installer typically creates.
#
# Idempotent: safe to run multiple times; exits 0 when there is nothing left to remove or only
# recoverable warnings (re-run if Helm releases remain due to transient errors).

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

INSTALLER_NS="${INSTALLER_NS:-tssc}"
DRY_RUN=0
DELETE_NAMESPACES=1
AGGRESSIVE=0
DELETE_OLM=0
MAX_PASSES=12
HELM_TIMEOUT="15m"
SKIP_HELM=0

usage() {
    sed -n '2,22p' "$0" | sed 's/^# \{0,1\}//'
    cat <<EOF

Options:
  -n, --installer-namespace NS   Installer / integration secret namespace (default: tssc)
  -d, --dry-run                  Print actions only
  -H, --helm-only                Only Helm uninstall; skip namespace deletion (-o still applies if set)
  -S, --skip-helm                Skip Helm uninstall; run -o and/or namespace deletion only
  -a, --aggressive               Also delete rhacs-operator and rhbk-operator namespaces
  -o, --olm                      Delete Operator Hub Subscriptions installed by TSSC charts
                                 (required because charts use helm.sh/resource-policy: keep)
  -p, --passes N                 Helm uninstall retry rounds (default: $MAX_PASSES)
  -t, --timeout DURATION         helm uninstall --timeout (default: $HELM_TIMEOUT)
  -h, --help                     This help

Environment:
  INSTALLER_NS                   Same as -n (default: tssc)

EOF
}

log() { printf '%s\n' "$*"; }

have() { command -v "$1" >/dev/null 2>&1; }

require_cmds() {
    local missing=()
    for c in oc helm jq; do
        have "$c" || missing+=("$c")
    done
    if ((${#missing[@]})); then
        log "[ERROR] Missing commands: ${missing[*]}"
        exit 1
    fi
}

# Normalize helm list JSON; empty or invalid input becomes [] so jq never aborts the script.
helm_list_json() {
    local raw
    raw="$(helm list -A -o json 2>/dev/null || true)"
    if [[ -z "$raw" ]] || ! echo "$raw" | jq -e . >/dev/null 2>&1; then
        echo '[]'
        return 0
    fi
    echo "$raw"
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
        -n | --installer-namespace)
            INSTALLER_NS="$2"
            shift 2
            ;;
        -d | --dry-run)
            DRY_RUN=1
            shift
            ;;
        -H | --helm-only)
            DELETE_NAMESPACES=0
            shift
            ;;
        -S | --skip-helm)
            SKIP_HELM=1
            shift
            ;;
        -a | --aggressive)
            AGGRESSIVE=1
            shift
            ;;
        -o | --olm)
            DELETE_OLM=1
            shift
            ;;
        --delete-olm-subscriptions)
            DELETE_OLM=1
            shift
            ;;
        -p | --passes)
            MAX_PASSES="$2"
            shift 2
            ;;
        -t | --timeout)
            HELM_TIMEOUT="$2"
            shift 2
            ;;
        -h | --help)
            usage
            exit 0
            ;;
        *)
            log "[ERROR] Unknown argument: $1"
            usage
            exit 1
            ;;
        esac
    done
}

# Prefer uninstalling leaf / application charts before subscriptions and global helpers.
# Higher score = uninstall earlier in each pass (helm list order otherwise arbitrary).
uninstall_weight() {
    local name="$1"
    case "$name" in
    tssc-dh | tssc-tpa | tssc-acs | tssc-tas | tssc-gitops | tssc-pipelines | tssc-iam) echo 90 ;;
    tssc-acs-test) echo 85 ;;
    tssc-pipelines-config) echo 84 ;;
    tssc-dh-access) echo 83 ;;
    *-subscriptions) echo 50 ;;
    *-openshift) echo 40 ;;
    tssc-app-namespaces | tssc-gitops-integrations | tssc-acs-integrations) echo 10 ;;
    *-pgsql) echo 70 ;;
    *) echo 60 ;;
    esac
}

helm_uninstall_all_tssc() {
    local pass releases_json n w name ns

    for ((pass = 1; pass <= MAX_PASSES; pass++)); do
        releases_json="$(helm_list_json)"
        n="$(echo "$releases_json" | jq '[.[] | select(.name | startswith("tssc-"))] | length')"
        if [[ "$n" -eq 0 ]]; then
            log "# Helm: no tssc-* releases left."
            return 0
        fi
        log "### Helm pass $pass/$MAX_PASSES ($n tssc-* releases)"

        while IFS=$'\t' read -r w name ns; do
            [[ -z "${name:-}" ]] && continue
            if [[ "$DRY_RUN" -eq 1 ]]; then
                log "[dry-run] helm uninstall $name -n $ns --wait --timeout $HELM_TIMEOUT"
                continue
            fi
            log "helm uninstall $name -n $ns"
            if ! helm uninstall "$name" -n "$ns" --wait --timeout "$HELM_TIMEOUT" 2>/dev/null; then
                log "# [WARNING] uninstall failed or partial: $name ($ns) — will retry next pass"
            fi
        done < <(
            echo "$releases_json" | jq -r '.[] | select(.name | startswith("tssc-")) | "\(.name)\t\(.namespace)"' |
                while IFS=$'\t' read -r name ns; do
                    w="$(uninstall_weight "$name")"
                    printf '%05d\t%s\t%s\n' "$w" "$name" "$ns"
                done | sort -rn
        )
        [[ "$DRY_RUN" -eq 1 ]] && return 0
    done

    releases_json="$(helm_list_json)"
    n="$(echo "$releases_json" | jq '[.[] | select(.name | startswith("tssc-"))] | length')"
    if [[ "$n" -gt 0 ]]; then
        log "# [WARNING] After $MAX_PASSES passes, $n tssc-* Helm release(s) remain (safe to re-run):"
        echo "$releases_json" | jq -r '.[] | select(.name | startswith("tssc-")) | "  - \(.name) (ns \(.namespace))"'
    fi
    return 0
}

# Subscription metadata.name + namespace as defined in bundles/*/tssc-*-subscriptions/values.yaml
# (union). Helm leaves these due to resource-policy: keep.
delete_olm_subscriptions() {
    local pair ns name
    # namespace<TAB>subscriptionName
    local -a pairs=(
        $'openshift-operators\topenshift-gitops-operator'
        $'openshift-operators\topenshift-pipelines-operator-rh'
        $'openshift-operators\trhtas-operator'
        $'openshift-operators\trhdh'
        $'rhbk-operator\trhbk-operator'
        $'rhacs-operator\trhacs-operator'
        $'tssc-tpa\trhtpa-operator'
    )

    if [[ "$DRY_RUN" -eq 1 ]]; then
        log "[dry-run] Would delete Operator Hub Subscriptions (if present):"
        for pair in "${pairs[@]}"; do
            ns="${pair%%$'\t'*}"
            name="${pair#*$'\t'}"
            log "    subscription $name -n $ns"
        done
        log "[dry-run] (Does not delete openshift-operators-wide OperatorGroups.)"
        return 0
    fi

    log "# Deleting TSSC Operator Hub Subscriptions (keep-policy leftovers)..."
    for pair in "${pairs[@]}"; do
        ns="${pair%%$'\t'*}"
        name="${pair#*$'\t'}"
        if ! oc get subscription "$name" -n "$ns" >/dev/null 2>&1; then
            continue
        fi
        log "oc delete subscription $name -n $ns"
        oc delete subscription "$name" -n "$ns" --ignore-not-found 2>/dev/null || true
    done

    # Dedicated operator namespaces: Subscription chart uses the same metadata.name for OperatorGroup.
    for pair in $'rhacs-operator\trhacs-operator' $'rhbk-operator\trhbk-operator' $'tssc-tpa\trhtpa-operator'; do
        ns="${pair%%$'\t'*}"
        og="${pair#*$'\t'}"
        if oc get operatorgroup "$og" -n "$ns" >/dev/null 2>&1; then
            log "oc delete operatorgroup $og -n $ns"
            oc delete operatorgroup "$og" -n "$ns" --ignore-not-found 2>/dev/null || true
        fi
    done
}

# Namespaces named exactly "tssc" or any "tssc-*" (installer project, bundle openshift.projects, overrides).
tssc_namespace_pattern() {
    grep -E '^tssc($|-)' || true
}

# Delete every cluster namespace matching ^tssc($|-). Do not skip names from an earlier fixed list —
# the previous straggler skipped bundle namespaces still present after a failed delete.
delete_all_tssc_prefixed_namespaces() {
    local ns
    log "# Delete all namespaces matching ^tssc($|-) (covers bundle defaults and custom names)..."
    while IFS= read -r ns; do
        [[ -z "$ns" ]] && continue
        log "oc delete namespace $ns --wait=false --ignore-not-found"
        oc delete namespace "$ns" --wait=false --ignore-not-found 2>/dev/null || true
    done < <(oc get ns -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null | tssc_namespace_pattern)
}

delete_namespace_if_present() {
    local name="$1"
    [[ -z "$name" ]] && return 0
    if ! oc get namespace "$name" >/dev/null 2>&1; then
        return 0
    fi
    log "oc delete namespace $name --wait=false --ignore-not-found"
    oc delete namespace "$name" --wait=false --ignore-not-found 2>/dev/null || true
}

delete_optional_namespaces() {
    local ns

    if [[ "$DRY_RUN" -eq 1 ]]; then
        log "[dry-run] Would delete installer namespaces if they do NOT match ^tssc($|-):"
        log "    $INSTALLER_NS ${INSTALLER_NS}-app (when custom project name)"
        [[ "$AGGRESSIVE" -eq 1 ]] && log "[dry-run] (aggressive) rhacs-operator rhbk-operator"
        log "[dry-run] Then delete every namespace matching ^tssc($|-) (tssc, tssc-keycloak, tssc-gitops, …)"
        return 0
    fi

    # Installer / app namespaces that are not matched by ^tssc($|-) (e.g. myproj / myproj-app).
    # No bash arrays here: empty "${arr[@]}" under bash 3.2 + set -u can error as unbound.
    if [[ ! "$INSTALLER_NS" =~ ^tssc($|-) ]]; then
        delete_namespace_if_present "$INSTALLER_NS"
    fi
    if [[ ! "${INSTALLER_NS}-app" =~ ^tssc($|-) ]]; then
        delete_namespace_if_present "${INSTALLER_NS}-app"
    fi

    if [[ "$AGGRESSIVE" -eq 1 ]]; then
        for ns in rhacs-operator rhbk-operator; do
            if ! oc get namespace "$ns" >/dev/null 2>&1; then
                continue
            fi
            log "oc delete namespace $ns --wait=false --ignore-not-found"
            oc delete namespace "$ns" --wait=false --ignore-not-found 2>/dev/null || true
        done
    fi

    delete_all_tssc_prefixed_namespaces

    log "# Waiting for removed namespaces to finish terminating (best-effort, 120s)..."
    local wait_start=$SECONDS
    while ((SECONDS - wait_start < 120)); do
        local pending=0
        while IFS= read -r ns; do
            [[ -z "$ns" ]] && continue
            oc get namespace "$ns" >/dev/null 2>&1 && pending=1
        done < <(oc get ns -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null | tssc_namespace_pattern)
        if [[ ! "$INSTALLER_NS" =~ ^tssc($|-) ]] && oc get namespace "$INSTALLER_NS" >/dev/null 2>&1; then
            pending=1
        fi
        if [[ ! "${INSTALLER_NS}-app" =~ ^tssc($|-) ]] && oc get namespace "${INSTALLER_NS}-app" >/dev/null 2>&1; then
            pending=1
        fi
        if [[ "$AGGRESSIVE" -eq 1 ]]; then
            for ns in rhacs-operator rhbk-operator; do
                oc get namespace "$ns" >/dev/null 2>&1 && pending=1
            done
        fi
        [[ "$pending" -eq 0 ]] && break
        sleep 3
    done
}

main() {
    parse_args "$@"
    require_cmds

    log "# TSSC cluster cleanup (installer namespace: $INSTALLER_NS)"
    log "# Current context: $(oc whoami --show-context 2>/dev/null || echo '?')"

    if [[ "$SKIP_HELM" -eq 0 ]]; then
        helm_uninstall_all_tssc
    else
        log "# Skipping Helm (-S/--skip-helm)"
    fi

    if [[ "$DELETE_OLM" -eq 1 ]]; then
        delete_olm_subscriptions
    fi

    if [[ "$DELETE_NAMESPACES" -eq 1 ]]; then
        delete_optional_namespaces
    else
        log "# Skipping namespace deletion (-H/--helm-only)"
    fi

    log "# Done."
    log "# Tip: if a namespace is stuck Terminating, see finalizer patching in hack/reset.sh"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
