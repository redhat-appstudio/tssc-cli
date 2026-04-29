#!/bin/bash

# Bitbucket repository cleanup script
# Run with --help for getting usage: 
# ./bitbucket-repos-cleanup.sh --help

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
    BITBUCKET_WORKSPACE - Bitbucket workspace name
    BITBUCKET_PROJECT   - Bitbucket project key
    
    Authentication (choose one):
    Option 1: Username + App Password
        BITBUCKET_USERNAME - Bitbucket username
        BITBUCKET_APP_PASSWORD - Bitbucket app password
    
    Option 2: Email + Access Token  
        BITBUCKET_EMAIL        - Your Bitbucket email address
        BITBUCKET_ACCESS_TOKEN - Bitbucket scoped API token

Optional environment variables:
    DAYS                - Number of days old repositories must be to qualify for cleanup (default: 14)
    REPO_NAME_REGEX     - Regex pattern to match repository names for cleanup (default is set to match the repository names created by tssc-test)

Optional arguments:
    -d, --dry-run       Enable dry run mode - no actual deletions
    -h, --help          Show this help message

Examples:
    # Using Username + App Password
    BITBUCKET_USERNAME=user BITBUCKET_APP_PASSWORD=app_pass BITBUCKET_WORKSPACE=workspace BITBUCKET_PROJECT=project ${0##*/} --dry-run
    BITBUCKET_USERNAME=user BITBUCKET_APP_PASSWORD=app_pass BITBUCKET_WORKSPACE=workspace BITBUCKET_PROJECT=project ${0##*/}
    
    # Using Email + Access Token
    BITBUCKET_EMAIL=user@example.com BITBUCKET_ACCESS_TOKEN=token BITBUCKET_WORKSPACE=workspace BITBUCKET_PROJECT=project ${0##*/} --dry-run
    BITBUCKET_EMAIL=user@example.com BITBUCKET_ACCESS_TOKEN=token BITBUCKET_WORKSPACE=workspace BITBUCKET_PROJECT=project ${0##*/}
"
}

bitbucket_cleanup() {
    export BITBUCKET_USERNAME="${BITBUCKET_USERNAME:-$(cat /usr/local/rhtap-cli-install/bitbucket-username 2>/dev/null || echo "")}"
    export BITBUCKET_APP_PASSWORD="${BITBUCKET_APP_PASSWORD:-$(cat /usr/local/rhtap-cli-install/bitbucket-app-password 2>/dev/null || echo "")}"
    export BITBUCKET_EMAIL="${BITBUCKET_EMAIL:-$(cat /usr/local/rhtap-cli-install/bitbucket-email 2>/dev/null || echo "")}"
    export BITBUCKET_ACCESS_TOKEN="${BITBUCKET_ACCESS_TOKEN:-$(cat /usr/local/rhtap-cli-install/bitbucket-access-token 2>/dev/null || echo "")}"
    export BITBUCKET_WORKSPACE="${BITBUCKET_WORKSPACE:-rhtap-test}"
    export BITBUCKET_PROJECT="${BITBUCKET_PROJECT:-RHTAP}"
    
    # Set up authentication for different auth methods
    if [[ -n "$BITBUCKET_USERNAME" && -n "$BITBUCKET_APP_PASSWORD" ]]; then
        # Option 1: Username + App Password
        AUTH_CREDS="$BITBUCKET_USERNAME:$BITBUCKET_APP_PASSWORD"
        echo "Using Username: $BITBUCKET_USERNAME + App Password authentication"

    elif [[ -n "$BITBUCKET_EMAIL" && -n "$BITBUCKET_ACCESS_TOKEN" ]]; then
        # Option 2: Email + Access Token
        AUTH_CREDS="$BITBUCKET_EMAIL:$BITBUCKET_ACCESS_TOKEN"
        echo "Using Email: $BITBUCKET_EMAIL + Access Token authentication"

    else
        echo "Error: No valid authentication method configured" >&2
        echo "Please set either:"
        echo "  - BITBUCKET_USERNAME and BITBUCKET_APP_PASSWORD (for app password auth)"
        echo "  - BITBUCKET_EMAIL and BITBUCKET_ACCESS_TOKEN (for access token auth)"
        exit 1
    fi
    
    # Calculate cutoff time
    cutoff_date=$(date -d "-${DAYS} days" --iso-8601)
    cutoff_time=$(date -d "$cutoff_date" +%s)
    
    echo "Checking Bitbucket workspace: $BITBUCKET_WORKSPACE"
    echo "Checking Bitbucket project: $BITBUCKET_PROJECT"
    echo "Cutoff date: $cutoff_date"
    echo ""
    
    # Fetch repositories with pagination (max pagelen=100 for Bitbucket).
    # Sort by updated_on (oldest first) to prioritize older repositories for cleanup.
    next_url="https://api.bitbucket.org/2.0/repositories/$BITBUCKET_WORKSPACE?q=project.key=\"$BITBUCKET_PROJECT\"&sort=updated_on&pagelen=100"
    all_repos=()
    while [[ -n "$next_url" ]]; do
        tmp_file=$(mktemp)
        http_status=$(curl -s -o "$tmp_file" -w "%{http_code}" -u "$AUTH_CREDS" -H "Accept: application/json" "$next_url")
        repos_page=$(<"$tmp_file")
        rm -f "$tmp_file"

        # Handle HTTP errors first.
        if [[ "$http_status" -lt 200 || "$http_status" -ge 300 ]]; then
            echo "Error: Bitbucket API returned HTTP $http_status" >&2
            if [[ -n "$repos_page" ]]; then
                echo "Response: $repos_page"
            fi
            return 1
        fi

        # Check for API errors (only if .error field exists and is not null).
        if echo "$repos_page" | jq -e '.error' >/dev/null 2>&1; then
            echo "Error fetching repositories: $(echo "$repos_page" | jq -r '.error.message')" >&2
            return 1
        fi

        # Check if we got a valid response with values array.
        if ! echo "$repos_page" | jq -e '.values' >/dev/null 2>&1; then
            echo "Error: Invalid response format from Bitbucket API" >&2
            echo "Response: $repos_page"
            return 1
        fi

        while IFS= read -r repo; do
            all_repos+=("$repo")
        done < <(echo "$repos_page" | jq -c '.values[]')

        next_url=$(echo "$repos_page" | jq -r '.next // empty')
    done

    if [[ ${#all_repos[@]} -gt 0 ]]; then
        repos=$(printf '%s\n' "${all_repos[@]}" | jq -s '{values: .}')
    else
        repos='{"values":[]}'
    fi

    repo_count=$(echo "$repos" | jq '.values | length')
    echo "Found $repo_count repositories"
    
    # Process repositories (using process substitution to avoid subshell)
    while read -r repo; do
        repo_name=$(echo "$repo" | jq -r '.name' 2>/dev/null)
        updated_on=$(echo "$repo" | jq -r '.updated_on' 2>/dev/null)

        # Convert updated_on to Unix timestamp for comparison
        updated_time=$(date -d "$updated_on" +%s)

        # Check if repository matches pattern and is old enough
        if [[ $repo_name =~ $repo_name_regex ]] && [[ $updated_time -lt $cutoff_time ]]; then
            if [[ "$dry_run" == "true" ]]; then
                echo "[DRY RUN] Would delete repository '$repo_name'. Last updated: $updated_on"
            else
                echo "Deleting repository '$repo_name'. Last updated: $updated_on"
                http_code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE \
                    -u "$AUTH_CREDS" \
                    "https://api.bitbucket.org/2.0/repositories/$BITBUCKET_WORKSPACE/$repo_name")
                
                if [[ "$http_code" -eq 204 ]]; then
                    echo "Repository '$repo_name' deleted successfully."
                else
                    echo "Failed to delete repository '$repo_name': $http_code"
                fi
            fi
        fi
    done <<< "$(echo "$repos" | jq -c '.values[]')"

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
    echo "Bitbucket Repository Cleanup Script"
    echo "==================================="  
    echo "Dry run: $dry_run"
    echo "Days threshold: $DAYS"
    echo ""
    
    bitbucket_cleanup
    
    echo "Bitbucket cleanup completed!"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
