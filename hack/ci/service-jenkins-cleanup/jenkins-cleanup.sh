#!/bin/bash

# Jenkins cleanup script
# Run with --help for getting usage:
# ./jenkins-cleanup.sh --help

set -o errexit
set -o nounset
set -o pipefail

# Common configuration
DAYS="${DAYS:-14}"

usage() {
    echo "
Usage:
    ${0##*/} [options]

Required environment variables:
    JENKINS_API_TOKEN   - Jenkins API token
    JENKINS_URL         - Jenkins server URL
    JENKINS_USERNAME    - Jenkins username

Optional environment variables:
    DAYS                - Number of days old jobs must be to qualify for cleanup (default: 14)

Optional arguments:
    -d, --dry-run       Enable dry run mode - no actual deletions
    -e, --empty-folders Delete empty folders
    -n, --no-builds     Delete jobs with no builds
    -v, --verbose       Enable verbose output
    -h, --help          Show this help message

Examples:
    # Dry run for Jenkins cleanup
    JENKINS_API_TOKEN=xxx JENKINS_URL=https://jenkins.example.com JENKINS_USERNAME=admin ${0##*/} --dry-run

    # Actually delete jobs
    JENKINS_API_TOKEN=xxx JENKINS_URL=https://jenkins.example.com JENKINS_USERNAME=admin ${0##*/}

    # Delete empty folders and jobs with no builds
    JENKINS_API_TOKEN=xxx JENKINS_URL=https://jenkins.example.com JENKINS_USERNAME=admin ${0##*/} --empty-folders --no-builds
"
}

# Global variables for folder processing
indent=""
delete_folder="true"
builds="false"
folder="false"
last_mod=0
check_date=0

item_cleanup() {
    local name="$1"
    local url="$2"

    # If NO_builds set, Force deletion if no builds by setting builds to true
    if [[ "${folder}" != "true" ]] && [[ "${no_builds}" == "true" ]] && [[ "${builds}" == "false" ]]; then
        builds="true"
        if [[ "$verbose" == "true" ]]; then
            echo "${indent}No Builds - Flag set deleting"
        fi
    fi

    # Clean up item according to settings
    if [[ "$delete_folder" == "true" ]] && { [[ "$builds" == "true" ]] || [[ "$folder" == "true" ]]; }; then
        local mod_date=""
        if [[ "$last_mod" != "0" ]]; then
            mod_date=$(TZ=UTC date -d @"$((last_mod/1000))")
        fi
        if [[ "$dry_run" == "false" ]]; then
            printf "%-10s %-60s %-40s\n" "Deleting" "${name}" "${mod_date}"
            curl -s -X POST -u "$JENKINS_USERNAME:$JENKINS_API_TOKEN" "${url}doDelete"
        else
            printf "%-10s %-60s %-40s\n" "[DRY RUN]" "${name}" "${mod_date}"
        fi
    else
        if [[ "$verbose" == "true" ]]; then
            echo "${indent}Skipping $name"
        fi
    fi
}

process_workflow() {
    local workflow_url="$1"
    indent="${indent}      "

    # Get list of builds and their timestamp
    local build_list
    build_list=$(curl -s -X POST -L -u "$JENKINS_USERNAME:$JENKINS_API_TOKEN" "${workflow_url}/api/json?tree=builds[number,timestamp]" --globoff | jq -r '.builds')

    local items
    items=$(echo "$build_list" | jq length)

    # If no builds skip directory - Do not delete
    if [[ "${items}" == "0" ]] && [[ "${builds}" != "true" ]]; then
        builds="false"
        if [[ "$verbose" == "true" ]]; then
            echo "${indent}No Builds"
        fi
        return
    fi

    # Loop through builds and if no recent builds mark for deletion
    builds="true"
    while read -r build_item; do
        local timestamp
        timestamp=$(echo "$build_item" | jq -r '.timestamp' 2>/dev/null)

        if (( timestamp > check_date )); then
            delete_folder="false"
            if [[ "$verbose" == "true" ]]; then
                echo "${indent}Skipping - Build has recently run"
            fi
            break
        fi

        if (( timestamp > last_mod )); then
            last_mod=$timestamp
        fi
    done < <(echo "$build_list" | jq -c '.[]')
}

process_folder() {
    local folder_url="$1"
    indent="${indent}      "

    # Get list of entries in directory and process
    local sub_dirs
    sub_dirs=$(curl -s --globoff -X POST -L -u "$JENKINS_USERNAME:$JENKINS_API_TOKEN" "${folder_url}api/json?tree=jobs[name,url]" | jq -r '.jobs')

    local items
    items=$(echo "$sub_dirs" | jq length)

    # If empty dir/folder - handle based on empty_folders flag
    if [[ "${items}" == "0" ]]; then
        if [[ "$empty_folders" == "true" ]]; then
            folder="true"
            if [[ "$verbose" == "true" ]]; then
                echo "${indent}Deleting - Empty folder"
            fi
        else
            delete_folder="false"
            if [[ "$verbose" == "true" ]]; then
                echo "${indent}Skipping - Empty folder"
            fi
        fi
        return
    fi
    process_list "${sub_dirs}"
}

process_list() {
    local list="$1"
    local items
    items=$(echo "$list" | jq length)

    if [[ "$verbose" == "true" ]]; then
        echo "${indent}items: $items"
    fi

    local i=0
    while read -r item; do
        i=$((i+1))

        local class name url
        class=$(echo "$item" | jq -r '._class' 2>/dev/null)
        name=$(echo "$item" | jq -r '.name' 2>/dev/null)
        url=$(echo "$item" | jq -r '.url' 2>/dev/null)

        if [[ "${indent}" == "" ]]; then
            delete_folder="true"
            builds="false"
            folder="false"
            last_mod=0
        fi

        if [[ "$class" == "com.cloudbees.hudson.plugins.folder.Folder" ]]; then
            # Process folders
            if [[ "$verbose" == "true" ]]; then
                echo "${indent}--- Processing item $i Folder $name ---"
            fi
            process_folder "${url}"
            if [[ "${indent}" == "      " ]]; then
                item_cleanup "$name" "$url"
            fi
        elif [[ "$class" == "org.jenkinsci.plugins.workflow.job.WorkflowJob" ]] || \
             [[ "$class" == "hudson.model.FreeStyleProject" ]]; then
            # Process workflow or freestyle Job
            if [[ "$verbose" == "true" ]]; then
                echo "${indent}--- Processing item $i Job $name"
            fi
            process_workflow "${url}"
            if [[ "${indent}" == "      " ]]; then
                item_cleanup "$name" "$url"
            fi
        else
            echo "${indent}WARNING: Unhandled class ${class}"
        fi

        indent=${indent::-6}
    done < <(echo "$list" | jq -c '.[]')
}

jenkins_cleanup() {
    export JENKINS_API_TOKEN="${JENKINS_API_TOKEN:-$(cat /usr/local/rhtap-cli-install/jenkins-api-token 2>/dev/null || echo "")}"
    export JENKINS_URL="${JENKINS_URL:-$(cat /usr/local/rhtap-cli-install/jenkins-url 2>/dev/null || echo "")}"
    export JENKINS_USERNAME="${JENKINS_USERNAME:-$(cat /usr/local/rhtap-cli-install/jenkins-username 2>/dev/null || echo "")}"

    # Validate required environment variables
    if [[ -z "$JENKINS_API_TOKEN" || -z "$JENKINS_URL" || -z "$JENKINS_USERNAME" ]]; then
        echo "Error: Required environment variables are not set" >&2
        echo "Please set:"
        echo "  - JENKINS_API_TOKEN (Jenkins API token)"
        echo "  - JENKINS_URL (Jenkins server URL)"
        echo "  - JENKINS_USERNAME (Jenkins username)"
        exit 1
    fi

    # Calculate cutoff time
    check_date=$(date -d "-${DAYS} days" +%s%3N)

    echo "Checking Jenkins server: $JENKINS_URL"
    echo "Jenkins username: $JENKINS_USERNAME"
    echo "Cutoff date: $(date -d "-${DAYS} days")"
    echo "Empty folders: $empty_folders"
    echo "No builds: $no_builds"
    echo "Verbose: $verbose"
    echo ""

    # Get list of top level entries in directory
    local dir_list
    dir_list=$(curl -s --globoff -X POST -L -u "$JENKINS_USERNAME:$JENKINS_API_TOKEN" "${JENKINS_URL}/api/json?tree=jobs[name,url]" | jq -r '.jobs')

    if [[ -z "$dir_list" || "$dir_list" == "null" ]]; then
        echo "Error: Failed to fetch Jenkins jobs or no jobs found" >&2
        return 1
    fi

    # Process entries in list searching for directories that do not have builds that have run in X number of days
    process_list "${dir_list}"
}

parse_args() {
    # Default configuration
    dry_run="false"
    empty_folders="false"
    no_builds="false"
    verbose="false"

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
            -e|--empty-folders)
                empty_folders="true"
                shift
                ;;
            -n|--no-builds)
                no_builds="true"
                shift
                ;;
            -v|--verbose)
                verbose="true"
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
    echo "Jenkins Cleanup Script"
    echo "==================================="
    echo "Dry run: $dry_run"
    echo "Days threshold: $DAYS"
    echo ""

    jenkins_cleanup

    echo "Jenkins cleanup completed!"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi

