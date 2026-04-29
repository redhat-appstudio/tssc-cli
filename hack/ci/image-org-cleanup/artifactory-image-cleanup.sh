#!/bin/bash

# JFrog Artifactory image cleanup script
# Run with --help for getting usage:
# ./artifactory-image-cleanup.sh --help

set -o errexit
set -o nounset
set -o pipefail

# Common configuration
DAYS="${DAYS:-14}"
repo_name_regex="${REPO_NAME_REGEX:-^[a-zA-Z0-9-]*(python|dotnet-basic|java-quarkus|go|nodejs|java-springboot)[a-zA-Z0-9-]*(-gitops)?\$}"

usage() {
    echo "
Usage:
    ${0##*/} [options]

Required environment variables:
    ARTIFACTORY_URL         - Artifactory server URL
    ARTIFACTORY_REPOSITORY  - Artifactory repository name
    ARTIFACTORY_API_TOKEN       - Artifactory API token

Optional environment variables:
    DAYS                    - Number of days old images must be to qualify for cleanup (default: 14)
    REPO_NAME_REGEX         - Regex pattern to match image folder names for cleanup (default matches names created by tssc-test)

Optional arguments:
    -d, --dry-run       Enable dry run mode - no actual deletions
    -h, --help          Show this help message

Examples:
    # Dry run for Artifactory cleanup (uses default URL and repository)
    ARTIFACTORY_API_TOKEN=xxx ${0##*/} --dry-run

    # Actually delete images
    ARTIFACTORY_API_TOKEN=xxx ${0##*/}
"
}

artifactory_cleanup() {
    export ARTIFACTORY_URL="${ARTIFACTORY_URL:-$(cat /usr/local/rhtap-cli-install/artifactory-url 2>/dev/null || echo "")}"
    export ARTIFACTORY_REPOSITORY="${ARTIFACTORY_REPOSITORY:-rhtap}"
    export ARTIFACTORY_API_TOKEN="${ARTIFACTORY_API_TOKEN:-$(cat /usr/local/rhtap-cli-install/artifactory-token 2>/dev/null || echo "")}"

    # Validate required environment variables
    if [[ -z "$ARTIFACTORY_API_TOKEN" || -z "$ARTIFACTORY_URL" ]]; then
        echo "Error: Required environment variables are not set" >&2
        echo "Please set:"
        echo "  - ARTIFACTORY_API_TOKEN (Artifactory API token)"
        echo "  - ARTIFACTORY_URL (Artifactory server URL)"
        exit 1
    fi

    AUTH_HEADER="Authorization: Bearer $ARTIFACTORY_API_TOKEN"

    # Calculate cutoff time
    now=$(date +%s)
    cutoff_time=$((now - DAYS * 24 * 60 * 60))
    cutoff_date=$(date -d "@$cutoff_time" --iso-8601)

    echo "Checking Artifactory repository: $ARTIFACTORY_REPOSITORY"
    echo "Artifactory URL: $ARTIFACTORY_URL"
    echo "Cutoff date: $cutoff_date"
    echo "Image name regex: $repo_name_regex"
    echo ""

    # Use AQL (Artifactory Query Language) with pagination to fetch all results.
    api_url="$ARTIFACTORY_URL/artifactory/api/search/aql"
    page_size="${ARTIFACTORY_PAGE_SIZE:-200}"
    offset=0
    all_items=()

    while :; do
        aql_query='items.find({"repo":"'"$ARTIFACTORY_REPOSITORY"'","type":"folder","depth":1}).include("name","created","modified").offset('"$offset"').limit('"$page_size"')'

        # Use a temp file to capture body, and get status code separately.
        tmp_file=$(mktemp)

        http_status=$(curl -s -o "$tmp_file" -w "%{http_code}" \
            -H "$AUTH_HEADER" \
            -H "Content-Type: text/plain" \
            -X POST \
            -d "$aql_query" \
            "$api_url")

        items_page=$(<"$tmp_file")
        rm -f "$tmp_file"

        # Handle HTTP errors first.
        if [[ "$http_status" -eq 401 ]]; then
            echo "Error: Authentication Failed (401). Please check your Artifactory credentials." >&2
            return 1
        elif [[ "$http_status" -eq 403 ]]; then
            echo "Error: Forbidden (403). Your credentials don't have access to this repository." >&2
            return 1
        elif [[ "$http_status" -eq 404 ]]; then
            echo "Error: Repository or API endpoint not found (404)." >&2
            return 1
        elif [[ "$http_status" -lt 200 || "$http_status" -ge 300 ]]; then
            echo "Error: API returned HTTP $http_status" >&2
            if [[ -n "$items_page" ]]; then
                echo "Response: $items_page"
            fi
            return 1
        fi

        # Check if we got a valid response with results array.
        if ! echo "$items_page" | jq -e '.results' >/dev/null 2>&1; then
            echo "Error: Invalid response format from Artifactory API" >&2
            echo "Response: ${items_page:-<empty>}"
            return 1
        fi

        while IFS= read -r item; do
            all_items+=("$item")
        done < <(echo "$items_page" | jq -c '.results[]')

        page_count=$(echo "$items_page" | jq '.results | length')
        if [[ "$page_count" -lt "$page_size" ]]; then
            break
        fi

        offset=$((offset + page_size))
    done

    if [[ ${#all_items[@]} -gt 0 ]]; then
        items=$(printf '%s\n' "${all_items[@]}" | jq -s '{results: .}')
    else
        items='{"results":[]}'
    fi

    item_count=$(echo "$items" | jq '.results | length')
    echo "Found $item_count items"

    # Process items
    while read -r item; do
        item_name=$(echo "$item" | jq -r '.name')

        if [[ ! "$item_name" =~ $repo_name_regex ]]; then
            continue
        fi

        item_modified=$(echo "$item" | jq -r '.modified // .created // empty')

        # Skip if no modified date
        if [[ -z "$item_modified" || "$item_modified" == "null" ]]; then
            continue
        fi

        # Convert modified time to Unix timestamp (Artifactory uses ISO 8601 format)
        item_modified_time=$(date -d "$item_modified" +%s 2>/dev/null || echo "0")

        # Image name matched regex and is old enough
        if [[ $item_modified_time -lt $cutoff_time ]]; then
            if [[ "$dry_run" == "true" ]]; then
                echo "[DRY RUN] Would delete item '$item_name'. Last modified: $item_modified"
            else
                echo "Deleting item '$item_name'. Last modified: $item_modified"
                delete_url="$ARTIFACTORY_URL/artifactory/$ARTIFACTORY_REPOSITORY/$item_name"

                http_code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE \
                    -H "$AUTH_HEADER" \
                    "$delete_url")

                if [[ "$http_code" -eq 204 || "$http_code" -eq 200 ]]; then
                    echo "Item '$item_name' deleted successfully."
                else
                    echo "Failed to delete item '$item_name': HTTP $http_code"
                fi
            fi
        fi
    done <<< "$(echo "$items" | jq -c '.results[]')"
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
    echo "Artifactory Repository Cleanup Script"
    echo "==================================="
    echo "Dry run: $dry_run"
    echo "Days threshold: $DAYS"
    echo ""

    artifactory_cleanup

    echo "Artifactory cleanup completed!"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
