#!/bin/bash

# Nexus image cleanup script
# Run with --help for getting usage:
# ./nexus-image-cleanup.sh --help

set -o errexit
set -o nounset
set -o pipefail

# Common configuration
DAYS="${DAYS:-14}"
# Allow / in names (e.g. rhtap/go-rtbcsyes); same stacks as other cleanup scripts.
repo_name_regex="${REPO_NAME_REGEX:-^[a-zA-Z0-9/-]*(python|dotnet-basic|java-quarkus|go|nodejs|java-springboot)[a-zA-Z0-9/-]*(-gitops)?\$}"

usage() {
    echo "
Usage:
    ${0##*/} [options]

Required environment variables:
    NEXUS_USERNAME      - Nexus username (default: admin)
    NEXUS_PASSWORD      - Nexus password
    NEXUS_URL           - Nexus server URL (default: https://nexus-ui-nexus.apps.rosa.rhtap-services.xmdt.p3.openshiftapps.com)
    NEXUS_REPOSITORY    - Nexus repository name (default: rhtap)

Optional environment variables:
    DAYS                - Number of days old images must be to qualify for cleanup (default: 14)
    REPO_NAME_REGEX     - Regex pattern to match component names for cleanup (default matches names created by tssc-test)

Optional arguments:
    -d, --dry-run       Enable dry run mode - no actual deletions
    -h, --help          Show this help message

Examples:
    # Dry run for Nexus cleanup (uses defaults for URL and repository)
    NEXUS_PASSWORD=xxx ${0##*/} --dry-run

    # Actually delete images
    NEXUS_PASSWORD=xxx ${0##*/}
"
}

nexus_cleanup() {
    export NEXUS_USERNAME="${NEXUS_USERNAME:-admin}"
    export NEXUS_PASSWORD="${NEXUS_PASSWORD:-$(cat /usr/local/rhtap-cli-install/nexus-password 2>/dev/null || echo "")}"
    export NEXUS_URL="${NEXUS_URL:-$(cat /usr/local/rhtap-cli-install/nexus-ui-url 2>/dev/null || echo "")}"
    export NEXUS_REPOSITORY="${NEXUS_REPOSITORY:-rhtap}"

    # Validate required environment variables
    if [[ -z "$NEXUS_USERNAME" || -z "$NEXUS_PASSWORD" || -z "$NEXUS_URL" ]]; then
        echo "Error: Required environment variables are not set" >&2
        echo "Please set:"
        echo "  - NEXUS_USERNAME (Nexus username)"
        echo "  - NEXUS_PASSWORD (Nexus password)"
        echo "  - NEXUS_URL (Nexus server URL)"
        exit 1
    fi

    AUTH_CREDS="$NEXUS_USERNAME:$NEXUS_PASSWORD"

    # Calculate cutoff time
    now=$(date +%s)
    cutoff_time=$((now - DAYS * 24 * 60 * 60))
    cutoff_date=$(date -d "@$cutoff_time" --iso-8601)

    echo "Checking Nexus repository: $NEXUS_REPOSITORY"
    echo "Nexus URL: $NEXUS_URL"
    echo "Cutoff date: $cutoff_date"
    echo ""

    # Fetch components from repository using Nexus REST API with pagination.
    next_token=""
    all_components=()

    while :; do
        api_url="$NEXUS_URL/service/rest/v1/components?repository=$NEXUS_REPOSITORY"
        if [[ -n "$next_token" ]]; then
            api_url="$api_url&continuationToken=$next_token"
        fi

        # Use a temp file to capture body, and get status code separately.
        tmp_file=$(mktemp)
        http_status=$(curl -s -o "$tmp_file" -w "%{http_code}" -u "$AUTH_CREDS" "$api_url")
        components_page=$(<"$tmp_file")
        rm -f "$tmp_file"

        # Handle HTTP errors first.
        if [[ "$http_status" -eq 401 ]]; then
            echo "Error: Authentication Failed (401). Please check your Nexus credentials." >&2
            return 1
        elif [[ "$http_status" -eq 403 ]]; then
            echo "Error: Forbidden (403). Your credentials don't have access to this repository." >&2
            return 1
        elif [[ "$http_status" -eq 404 ]]; then
            echo "Error: Repository not found (404). Please check NEXUS_REPOSITORY name." >&2
            return 1
        elif [[ "$http_status" -lt 200 || "$http_status" -ge 300 ]]; then
            echo "Error: API returned HTTP $http_status" >&2
            if [[ -n "$components_page" ]]; then
                echo "Response: $components_page"
            fi
            return 1
        fi

        # Check if we got a valid response with items array.
        if ! echo "$components_page" | jq -e '.items' >/dev/null 2>&1; then
            echo "Error: Invalid response format from Nexus API" >&2
            echo "Response: ${components_page:-<empty>}"
            return 1
        fi

        while IFS= read -r component; do
            all_components+=("$component")
        done < <(echo "$components_page" | jq -c '.items[]')

        next_token=$(echo "$components_page" | jq -r '.continuationToken // empty')
        if [[ -z "$next_token" ]]; then
            break
        fi
    done

    if [[ ${#all_components[@]} -gt 0 ]]; then
        components=$(printf '%s\n' "${all_components[@]}" | jq -s '{items: .}')
    else
        components='{"items":[]}'
    fi

    component_count=$(echo "$components" | jq '.items | length')
    echo "Found $component_count components"

    # Process components
    while read -r component; do
        component_name=$(echo "$component" | jq -r '.name')
        component_id=$(echo "$component" | jq -r '.id')

        if [[ ! "$component_name" =~ $repo_name_regex ]]; then
            continue
        fi

        last_modified=$(echo "$component" | jq -r '[.assets[].lastModified // empty] | map(select(. != null and . != "")) | sort | last // empty')

        # Skip if no last_modified date
        if [[ -z "$last_modified" || "$last_modified" == "null" ]]; then
            continue
        fi

        # Convert last_modified to Unix timestamp (Nexus uses ISO 8601 format)
        last_modified_time=$(date -d "$last_modified" +%s 2>/dev/null || echo "0")

        # Component name matched regex and is old enough
        if [[ $last_modified_time -lt $cutoff_time ]]; then
            if [[ "$dry_run" == "true" ]]; then
                echo "[DRY RUN] Would delete component '$component_name'. Last modified: $last_modified"
            else
                echo "Deleting component '$component_name'. Last modified: $last_modified"
                delete_url="$NEXUS_URL/service/rest/v1/components/$component_id"
                http_code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE \
                    -u "$AUTH_CREDS" \
                    "$delete_url")

                if [[ "$http_code" -eq 204 || "$http_code" -eq 200 ]]; then
                    echo "Component '$component_name' deleted successfully."
                else
                    echo "Failed to delete component '$component_name': HTTP $http_code"
                fi
            fi
        fi
    done <<< "$(echo "$components" | jq -c '.items[]')"
}

parse_args() {
    # Default configuration
    dry_run="false"

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -h|--help)
                usage
                exit 0
                ;;
            -d|--dry-run)
                dry_run="true"
                shift
                ;;
            *)
                echo "Error: Invalid command syntax" >&2
                usage
                exit 1
                ;;
        esac
    done
}

main() {
    parse_args "$@"

    echo "==================================="
    echo "Nexus Repository Cleanup Script"
    echo "==================================="
    echo "Dry run: $dry_run"
    echo "Days threshold: $DAYS"
    echo ""

    nexus_cleanup

    echo "Nexus cleanup completed!"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
