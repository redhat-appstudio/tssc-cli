#!/bin/bash

# Quay image cleanup script
# Run with --help for getting usage:
# ./quay-image-cleanup.sh --help

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
    QUAY_API_TOKEN      - Quay API token (OAuth access token)
    QUAY_ORGANIZATION   - Quay organization name

Optional environment variables:
    QUAY_URL            - Quay server URL (default: https://quay.io)
    DAYS                - Number of days old repositories must be to qualify for cleanup (default: 14)
    REPO_NAME_REGEX     - Regex pattern to match repository names for cleanup (default matches names created by tssc-test)

Optional arguments:
    -d, --dry-run       Enable dry run mode - no actual deletions
    -h, --help          Show this help message

Examples:
    # Dry run for Quay cleanup
    QUAY_API_TOKEN=xxx QUAY_ORGANIZATION=myorg ${0##*/} --dry-run

    # Actually delete repositories (default behavior)
    QUAY_API_TOKEN=xxx QUAY_ORGANIZATION=myorg ${0##*/}
"
}

quay_cleanup() {
    export QUAY_API_TOKEN="${QUAY_API_TOKEN:-$(cat /usr/local/rhtap-cli-install/quay-api-token 2>/dev/null || echo "")}"
    export QUAY_ORGANIZATION="${QUAY_ORGANIZATION:-rhtap_qe}"
    export QUAY_URL="${QUAY_URL:-https://quay.io}"

    # Validate required environment variables
    if [[ -z "$QUAY_API_TOKEN" || -z "$QUAY_ORGANIZATION" ]]; then
        echo "Error: Required environment variables are not set" >&2
        echo "Please set:"
        echo "  - QUAY_API_TOKEN (Quay API token)"
        echo "  - QUAY_ORGANIZATION (Quay organization name)"
        exit 1
    fi

    AUTH_HEADER="Authorization: Bearer $QUAY_API_TOKEN"

    # Calculate cutoff time
    now=$(date +%s)
    cutoff_time=$((now - DAYS * 24 * 60 * 60))
    cutoff_date=$(date -d "@$cutoff_time" --iso-8601)

    echo "Checking Quay organization: $QUAY_ORGANIZATION"
    echo "Quay URL: $QUAY_URL"
    echo "Cutoff date: $cutoff_date"
    echo "Repository name regex: $repo_name_regex"
    echo ""

    # Fetch repositories from organization with pagination support.
    next_page=""
    all_repos=()

    while :; do
        api_url="$QUAY_URL/api/v1/repository?namespace=$QUAY_ORGANIZATION"
        if [[ -n "$next_page" ]]; then
            api_url="$api_url&next_page=$next_page"
        fi

        # Use a temp file to capture body, and get status code separately.
        tmp_file=$(mktemp)
        http_status=$(curl -s -o "$tmp_file" -w "%{http_code}" -H "$AUTH_HEADER" "$api_url")
        repos=$(<"$tmp_file")
        rm -f "$tmp_file"

        # Handle HTTP errors first.
        if [[ "$http_status" -eq 401 ]]; then
            echo "Error: Authentication Failed (401). Please check your Quay token." >&2
            return 1
        elif [[ "$http_status" -eq 403 ]]; then
            echo "Error: Forbidden (403). Your token doesn't have access to this organization." >&2
            return 1
        elif [[ "$http_status" -eq 404 ]]; then
            echo "Error: Organization not found (404). Please check QUAY_ORGANIZATION name." >&2
            return 1
        elif [[ "$http_status" -lt 200 || "$http_status" -ge 300 ]]; then
            echo "Error: API returned HTTP $http_status" >&2
            if [[ -n "$repos" ]]; then
                echo "Response: $repos"
            fi
            return 1
        fi

        # Check for API errors.
        if echo "$repos" | jq -e '.error_message' >/dev/null 2>&1; then
            echo "Error fetching repositories: $(echo "$repos" | jq -r '.error_message')" >&2
            return 1
        fi

        # Check if we got a valid response with repositories array.
        if ! echo "$repos" | jq -e '.repositories' >/dev/null 2>&1; then
            echo "Error: Invalid response format from Quay API" >&2
            echo "Response: ${repos:-<empty>}"
            return 1
        fi

        # Append current page repositories into the accumulator.
        while IFS= read -r repo; do
            all_repos+=("$repo")
        done < <(echo "$repos" | jq -c '.repositories[]')

        next_page=$(echo "$repos" | jq -r '.next_page // empty')
        if [[ -z "$next_page" ]]; then
            break
        fi
    done

    if [[ ${#all_repos[@]} -gt 0 ]]; then
        repos=$(printf '%s\n' "${all_repos[@]}" | jq -s '{repositories: .}')
    else
        repos='{"repositories":[]}'
    fi

    repo_count=$(echo "$repos" | jq '.repositories | length')
    echo "Found $repo_count repositories"

    # Process repositories
    while read -r repo; do
        repo_name=$(echo "$repo" | jq -r '.name')

        if [[ ! "$repo_name" =~ $repo_name_regex ]]; then
            continue
        fi

        # Fetch repository details to get tags (list API doesn't include last_modified)
        repo_detail_url="$QUAY_URL/api/v1/repository/$QUAY_ORGANIZATION/$repo_name"
        repo_detail=$(curl -s -H "$AUTH_HEADER" "$repo_detail_url")

        # Get the most recent tag's last_modified date
        last_modified=$(echo "$repo_detail" | jq -r '[.tags[].last_modified // empty] | map(select(. != null and . != "")) | sort | last // empty')

        # Skip if no last_modified date found in any tag
        if [[ -z "$last_modified" || "$last_modified" == "null" ]]; then
            continue
        fi

        # Convert RFC 2822 date format to Unix timestamp
        # Format: "Mon, 09 Jun 2025 11:17:19 -0000"
        last_modified_time=$(date -d "$last_modified" +%s 2>/dev/null || echo "0")
        last_modified_date=$(date -d "$last_modified" --iso-8601 2>/dev/null || echo "$last_modified")

        # Check if old enough
        if [[ $last_modified_time -ge $cutoff_time ]]; then
            continue
        fi

        # Repository name matched regex and is old enough
        if [[ "$dry_run" == "true" ]]; then
            echo "[DRY RUN] Would delete repository '$repo_name'. Last modified: $last_modified_date"
        else
            echo "Deleting repository '$repo_name'. Last modified: $last_modified_date"
            delete_url="$QUAY_URL/api/v1/repository/$QUAY_ORGANIZATION/$repo_name"
            http_code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE \
                -H "$AUTH_HEADER" \
                "$delete_url")

            if [[ "$http_code" -eq 204 || "$http_code" -eq 200 ]]; then
                echo "Repository '$repo_name' deleted successfully."
            else
                echo "Failed to delete repository '$repo_name': HTTP $http_code"
            fi
        fi
    done <<< "$(echo "$repos" | jq -c '.repositories[]')"
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
    echo "Quay Repository Cleanup Script"
    echo "==================================="
    echo "Dry run: $dry_run"
    echo "Days threshold: $DAYS"
    echo ""

    quay_cleanup

    echo "Quay cleanup completed!"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
