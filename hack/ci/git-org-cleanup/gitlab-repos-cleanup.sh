#!/bin/bash

# GitLab repository cleanup script
# Run with --help for getting usage: 
# ./gitlab-repos-cleanup.sh --help

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
    GITLAB_TOKEN        - GitLab personal access token
    GITLAB_GROUP        - GitLab group ID

Optional environment variables:
    GITLAB_URL          - GitLab instance URL (default: https://gitlab.com)
    DAYS                - Number of days old repositories must be to qualify for cleanup (default: 14)
    REPO_NAME_REGEX     - Regex pattern to match repository names for cleanup (default is set to match the repository names created by tssc-test)

Optional arguments:
    -d, --dry-run       Enable dry run mode - no actual deletions
    -h, --help          Show this help message

Examples:
    # Dry run for GitLab cleanup
    GITLAB_TOKEN=xxx GITLAB_GROUP=123 ${0##*/} --dry-run

    # Actually delete repositories (default behavior)
    GITLAB_TOKEN=xxx GITLAB_GROUP=123 ${0##*/}
"
}

gitlab_cleanup() {
    export GITLAB_TOKEN="${GITLAB_TOKEN:-$(cat /usr/local/rhtap-cli-install/gitlab_token 2>/dev/null || echo "")}"
    export GITLAB_GROUP="${GITLAB_GROUP:-$(cat /usr/local/rhtap-cli-install/gitlab-group 2>/dev/null || echo "")}"
    export GITLAB_URL="${GITLAB_URL:-https://gitlab.com}"
    
    # Validate required environment variables
    if [[ -z "$GITLAB_TOKEN" || -z "$GITLAB_GROUP" ]]; then
        echo "Error: Required environment variables are not set" >&2
        echo "Please set:"
        echo "  - GITLAB_TOKEN (GitLab personal access token)"
        echo "  - GITLAB_GROUP (GitLab group ID)"
        exit 1
    fi
    
    AUTH_HEADER="PRIVATE-TOKEN: $GITLAB_TOKEN"
    
    
    # Calculate cutoff time
    cutoff_date=$(date -d "-${DAYS} days" --iso-8601)
    
    echo "Checking GitLab group: $GITLAB_GROUP"
    echo "GitLab URL: $GITLAB_URL"
    echo "Cutoff date: $cutoff_date"
    echo ""
    
    # Fetch projects from group (sorted by creation date, oldest first)
    projects=$(curl -s -H "$AUTH_HEADER" \
        "$GITLAB_URL/api/v4/groups/$GITLAB_GROUP/projects?per_page=200&order_by=created_at&sort=asc")
    
    if echo "$projects" | jq -e '.message or .error' 2>/dev/null ; then
        echo "Error fetching projects: $projects" >&2
        return 1
    fi
    
    # Process projects (using process substitution to avoid subshell)
    while read -r project; do
        project_name=$(echo "$project" | jq -r '.name' 2>/dev/null)
        project_id=$(echo "$project" | jq -r '.id' 2>/dev/null)
        last_activity=$(echo "$project" | jq -r '.last_activity_at' 2>/dev/null)
        
        # Convert last activity to Unix timestamp for comparison
        last_activity_time=$(date -d "$last_activity" +%s)
        cutoff_time=$(date -d "$cutoff_date" +%s)
        
        # Check if project matches pattern and is old enough
        if [[ $project_name =~ $repo_name_regex ]] && [[ $last_activity_time -lt $cutoff_time ]]; then
            if [[ "$dry_run" == "true" ]]; then
                echo "[DRY RUN] Would delete project '$project_name' (ID: $project_id). Last activity: $last_activity"
            else
                echo "Deleting project '$project_name' (ID: $project_id). Last activity: $last_activity"
                http_code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE \
                    -H "$AUTH_HEADER" \
                    "$GITLAB_URL/api/v4/projects/$project_id")
                
                if [[ "$http_code" -eq 202 ]] || [[ "$http_code" -eq 204 ]]; then
                    echo "Project '$project_name' deleted successfully."
                else
                    echo "Failed to delete project '$project_name': HTTP $http_code"
                fi
            fi
        fi
    done <<< "$(echo "$projects" | jq -c '.[]')"
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
    echo "GitLab Repository Cleanup Script"
    echo "==================================="  
    echo "Dry run: $dry_run"
    echo "Days threshold: $DAYS"
    echo ""
        
    gitlab_cleanup

    echo "GitLab cleanup completed!"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
