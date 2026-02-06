#!/bin/bash

# GitHub repository cleanup script
# Run with --help for getting usage:
# ./github-repos-cleanup.sh --help

set -o errexit
set -o nounset
set -o pipefail

# Common configuration
DAYS="${DAYS:-14}"
repo_name_regex="${REPO_NAME_REGEX:-^[a-z0-9-]*(python|dotnet-basic|java-quarkus|go|nodejs|java-springboot)[a-z0-9-]*(-gitops)?$}"

usage() {
    echo "
Usage:
    ${0##*/} [options]

Required environment variables:
    GITHUB_TOKEN        - GitHub personal access token
    GITHUB_ORG          - GitHub organization name

Optional environment variables:
    DAYS                - Number of days old repositories must be to qualify for cleanup (default: 14)
    REPO_NAME_REGEX     - Regex pattern to match repository names for cleanup (default is set to match the repository names created by tssc-test)

Optional arguments:
    -d, --dry-run       Enable dry run mode - no actual deletions
    -h, --help          Show this help message

Examples:
    # Dry run for GitHub cleanup
    GITHUB_TOKEN=xxx GITHUB_ORG=myorg ${0##*/} --dry-run

    # Actually delete repositories (default behavior)
    GITHUB_TOKEN=xxx GITHUB_ORG=myorg ${0##*/}
"
}

github_cleanup() {
    export GITHUB_TOKEN="${GITHUB_TOKEN:-$(cat /usr/local/rhtap-cli-install/github_token 2>/dev/null || echo "")}"
    export GITHUB_ORG="${GITHUB_ORG:-rhtap-rhdh-qe}"
    
    # Validate required environment variables
    if [[ -z "$GITHUB_TOKEN" || -z "$GITHUB_ORG" ]]; then
        echo "Error: Required environment variables are not set" >&2
        echo "Please set:"
        echo "  - GITHUB_TOKEN (GitHub personal access token)"
        echo "  - GITHUB_ORG (GitHub organization name)"
        exit 1
    fi
    
    AUTH_HEADER="Authorization: token $GITHUB_TOKEN"
    
    # Calculate cutoff time
    now=$(date +%s)
    cutoff_time=$((now - DAYS * 24 * 60 * 60))
    
    echo "Checking GitHub organization: $GITHUB_ORG"
    echo "Cutoff date: $(date -d "@$cutoff_time")"
    
    # Fetch repositories (sorted by creation date, oldest first)
    repos=$(curl -s -X GET -H "$AUTH_HEADER" "https://api.github.com/orgs/$GITHUB_ORG/repos?per_page=200&sort=created&direction=asc")
    if echo "$repos" | jq -e '.status' >/dev/null 2>&1; then
        echo "Error fetching repositories: $repos" >&2
        return 1
    fi
    
    # Process repositories
    while read -r repo; do
        repo_name=$(echo "$repo" | jq -r '.name' 2>/dev/null)
        last_push=$(echo "$repo" | jq -r '.pushed_at' 2>/dev/null)
        
        # Convert the last push time to Unix timestamp
        last_push_time=$(date -d "$last_push" +%s)
        
        # Check if repository matches pattern and is old enough
        if [[ $repo_name =~ $repo_name_regex ]] && [[ $last_push_time -lt $cutoff_time ]]; then
            if [[ "$dry_run" == "true" ]]; then
                echo "[DRY RUN] Would delete repository '$repo_name'. Last updated: $last_push"
            else
                echo "Deleting repository '$repo_name'. Last updated: $last_push"
                http_code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE \
                    -H "$AUTH_HEADER" \
                    "https://api.github.com/repos/$GITHUB_ORG/$repo_name")
                
                if [[ "$http_code" -eq 204 ]]; then
                    echo "Repository '$repo_name' deleted successfully."
                else
                    echo "Failed to delete repository '$repo_name': HTTP $http_code"
                fi
            fi
        fi
    done <<< "$(echo "$repos" | jq -c '.[]')"
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
    echo "GitHub Repository Cleanup Script"
    echo "==================================="  
    echo "Dry run: $dry_run"
    echo "Days threshold: $DAYS"
    echo ""

    github_cleanup

    echo "GitHub cleanup completed!"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
