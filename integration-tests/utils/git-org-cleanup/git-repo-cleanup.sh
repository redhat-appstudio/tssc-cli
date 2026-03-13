#!/bin/bash

# Set dry_run to true for not deleting repos, it will provide list of repos to delete
dry_run="${dry_run:-true}"

# Set the GitHub Token and Organization name
export GITHUB_TOKEN="$GITHUB_ORG_TOKEN"
export GITHUB_ORG="$GITHUB_ORG_NAME"

# Headers for authentication
AUTH_HEADER="Authorization: token $GITHUB_TOKEN"

# Get the current time and the time 2 weeks ago
now=$(date +%s)
cutoff_time=$((now - 14 * 24 * 60 * 60))

# Set the regex string to match with e2e-tests repos name
repo_name_regex="^[a-z0-9-]+-(python|dotnet-basic|java-quarkus|go|nodejs|java-springboot)(-[0-9a-z]+)?(-gitops)?$"
# Fetch the list of repositories from GitHub API
repos=$(curl -s -X GET -H "$AUTH_HEADER" "https://api.github.com/orgs/$GITHUB_ORG/repos?per_page=100&sort=name")
if [ "$(echo $repos | jq -r .status 2>/dev/null)" ]; then
    echo "Error Fetching repositories '$repos' "
    exit 1
fi

echo "$repos" | jq -c '.[]' | while read -r repo; do
    repo_name=$(echo $repo | jq -r '.name')
    last_push=$(echo $repo | jq -r '.pushed_at')

    # Convert the last push time to Unix timestamp
    last_push_time=$(date -d "$last_push" +%s)

    # If the repository hasn't been updated in the last 2 weeks, consider it for cleanup
    if [[ $repo_name =~ $repo_name_regex ]] && [ $last_push_time -lt $cutoff_time ]; then
        echo "Deleting repository '$repo_name'. Last updated at '$last_push' "

        if [[ "$dry_run" != "true" ]]; then
            delete_response=$(curl -s -X DELETE \
                -H "$AUTH_HEADER" \
                "https://api.github.com/repos/$GITHUB_ORG/$repo_name")

            if [ -z "$delete_response" ]; then
                echo "Repository '$repo_name' deleted."
            else
                echo "Failed to delete repository '$repo_name'\n$delete_response"
            fi
        fi

    fi
done
