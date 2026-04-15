#!/bin/bash

# Set dry_run to true for not deleting jenkins job directories, it will provide list of directories/job to delete
dry_run="${dry_run:-true}"

# Set verbose add info displayed
VERBOSE="${verbose:-false}"

# Set Deletion of empty folders and jobs with no builds
EMPTY_FOLDERS="${empty_folders:-false}"
NO_BUILDS="${no_builds:-false}"

# Last modification should be old than this number of DAYS
DAYS="${DAYS:-14}"

# Indent to line output
INDENT=""

# Set JENINS_API_TOKEN, JENKINS_URL and JENKINS_USERNAME
export JENKINS_API_TOKEN="${JENKINS_API_TOKEN}"
export JENKINS_URL="${JENKINS_URL:-https://jenkins-jenkins.apps.rosa.rhtap-services.xmdt.p3.openshiftapps.com}"
export JENKINS_USERNAME="${JENKINS_USERNAME:-cluster-admin-admin-edit-view}"

help_text() {
    echo " "
    echo "$0 - Cleans up old job directories that have no builds that have run "
    echo "     in X number of days"
    echo " "
    echo "$0 [options]"
    echo " "
    echo "Note: Environment variables JENKINS_API_TOKEN is required to be set."
    echo "      Must set environment variables JENKINS_USERNAME and JENKINS_URL "
    echo "           if not using defaults of 'cluster-admin-admin-edit-view' and"
    echo "           'https://jenkins-jenkins.apps.rosa.rhtap-services.xmdt.p3.openshiftapps.com'"
    echo " "
    echo "options:"
    echo "-h, --help                  Show brief help"
    echo "-d, --dry_run=dry_run       No actions actually performed, List of directories to remove. Valid values ['true', 'false'] Default: true"
    echo "-o, --older=DAYS            Specify number of days old directories last modified, Default: 14"
    echo "-e, --empty_folders         Delete empty folders. Default: false"
    echo "-n, --no_builds             Delete if contains no builds. Default: false"
    echo "-v, --verbose               Will output verbose info about processing. Default: false"
}

short_opt() {
    local VAR_NAME="$1"
    declare -n VAR_REF="$VAR_NAME"
    shift
    shift
    if [ $# -gt 0 ] && [[ ! "$1" =~ ^-.* ]] ; then
          VAR_REF=$1
    else
          echo "ERROR: Invalid syntax. Option specified but no $VAR_NAME value supplied"
          help_text
          exit 1
    fi
    shift
}

long_opt() {
    local VAR_NAME="$1"
    declare -n VAR_REF="$VAR_NAME"
    shift
    VAR_REF=`echo $1 | sed -e 's/^[^=]*=//g'`
    shift
}

verify_syntax() {
    while test $# -gt 0; do
      case "$1" in
        -h|--help)
          help_text
          exit 0
          ;;
        -o)
          short_opt "DAYS" $@
          shift 2
          ;;
        --older*)
          long_opt "DAYS" $@
          shift
          ;;
        -d)
          short_opt "dry_run" $@
          shift 2
          ;;
        --dry_run*)
          long_opt "dry_run" $@
          shift
          ;;
        -e|--empty_folders)
          shift
          EMPTY_FOLDERS=true
          ;;
        -n|--no_builds)
          shift
          NO_BUILDS=true
          ;;
        -v|--verbose)
          shift
          VERBOSE=true
          ;;
        *)
          echo "Error: Invalid command syntax"
          help_text
          exit 1
          ;;
      esac
    done

    if [[ "$dry_run" != "true" ]] && [[ "$dry_run" != "false" ]]; then
        echo "Error: -d/--dry_run must be set to 'true' or 'false'"
        exit 1
    fi

    # Verify creds specified
    if [[ -z "${JENKINS_API_TOKEN}" ]]; then
        echo "ERROR: JENKINS_API_TOKEN must be set."
        exit 1
    fi
}

item_cleanup() {
    # If NO_BUILDS set, Force deletion if no builds by setting BUILDS to true
    if [ "${FOLDER}" != "true" ] && [ "${NO_BUILDS}" == "true" ] && [ "${BUILDS}" == "false" ]; then
        BUILDS="true"
        if [[ "$VERBOSE" == "true" ]]; then
            echo "${INDENT}No Builds - Flag set deleting"
            echo " "
        fi
    fi

    # Clean up item according to settings
    if [ "$DELETE_FOLDER" == "true" ] && ([ "$BUILDS" == "true" ] || [ "$FOLDER" == "true" ]); then
        if [ "$LAST_MOD" == "0" ]; then
            MOD_DATE=""
        else
            MOD_DATE=`TZ=UTC date -d @"$((LAST_MOD/1000))"`
        fi
        if [[ "$dry_run" == "false" ]]; then
            printf "%-10s %-60s %-40s\n" "Deleting" "${NAME}" "${MOD_DATE}"
            curl -s -X POST -u "$JENKINS_USERNAME:$JENKINS_API_TOKEN" "${URL}doDelete"
        else
           printf "%-10s %-60s %-40s\n" "Delete" "${NAME}" "${MOD_DATE}"
        fi
    else
        if [[ "$VERBOSE" == "true" ]]; then
            echo "${INDENT}Skipping $NAME"
            echo " "
        fi
    fi
}

process_list() {
    # Setup  local vars
    local CLASS
    local NAME
    local URL
    local ITEMS
    local i

    # Get number of items in list
    ITEMS="$(echo "$1" | jq length)"

    if [[ "$VERBOSE" == "true" ]]; then
        echo "${INDENT}ITEMS: $ITEMS"
    fi

    # Process each item and handle as directory or workflow
    i=0
    while read item; do
        i=$((i+1))

        CLASS=$(echo $item | jq -r '._class' 2>/dev/null)
        NAME=$(echo $item | jq -r '.name' 2>/dev/null)
        URL=$(echo $item | jq -r '.url' 2>/dev/null)

        if [ "${INDENT}" == "" ]; then
            DELETE_FOLDER="true"
            BUILDS="false"
            FOLDER="false"
            LAST_MOD=0
        fi

        if [ "$CLASS" == "com.cloudbees.hudson.plugins.folder.Folder" ]; then
            # Process folders
            if [[ "$VERBOSE" == "true" ]]; then
                echo "${INDENT}--- Processing item $i Folder $NAME ---"
            fi
            process_folder "${URL}"
            if [ "${INDENT}" == "      " ]; then
                item_cleanup
            fi
        elif [ "$CLASS" == "org.jenkinsci.plugins.workflow.job.WorkflowJob" ]; then
            # Process workflow Job
            if [[ "$VERBOSE" == "true" ]]; then
                echo "${INDENT}--- Processing item $i Job $NAME"
                echo " "
            fi
            process_workflow "${URL}"
            if [ "${INDENT}" == "      " ]; then
                item_cleanup
            fi
        elif [ "$CLASS" == "hudson.model.FreeStyleProject" ]; then
            # Process Free Style Job
            if [[ "$VERBOSE" == "true" ]]; then
                echo "${INDENT}--- Processing item $i Job $NAME"
                echo " "
            fi
            process_workflow "${URL}"
            if [ "${INDENT}" == "      " ]; then
                item_cleanup
            fi
        else
            echo "${INDENT}WARNING: Unhandled CLASS ${CLASS}"
        fi

        INDENT=${INDENT::-6}
        if [[ "$VERBOSE" == "true" ]]; then
            echo ""
        fi

    done < <(echo $1 | jq -c '.[]')
}

process_folder() {
    local sub_dirs
    INDENT="${INDENT}      "

    # Get list of entries in directory and process
    sub_dirs=`curl -s --globoff -X POST -L -u "$JENKINS_USERNAME:$JENKINS_API_TOKEN" ${1}api/json?tree=jobs[name,url] | jq -r .jobs`

    local ITEMS
    ITEMS="$(echo "$sub_dirs" | jq length)"

    # If empty dir/folder - Do not delete
    if [ "${ITEMS}" == "0" ]; then
        if [[ "$EMPTY_FOLDERS" == "true" ]]; then
            FOLDER="true"
            if [[ "$VERBOSE" == "true" ]]; then
                echo "${INDENT}Deleting - Empty folder"
            fi
        else
            DELETE_FOLDER="false"
            if [[ "$VERBOSE" == "true" ]]; then
                echo "${INDENT}Skipping - Empty folder"
            fi
        fi
        return
    fi
    process_list "${sub_dirs}"
}

process_workflow() {
    local build_list
    local j
    INDENT="${INDENT}      "

    # Get list of builds and their timestamp
    build_list=`curl -s -X POST -L -u "$JENKINS_USERNAME:$JENKINS_API_TOKEN" $1/api/json?tree=builds[number,timestamp] --globoff | jq -r .builds`

    local ITEMS
    ITEMS="$(echo "$build_list" | jq length)"

    # If no builds skip directory - Do not delete
    if [ "${ITEMS}" == "0" ] && [ "${BUILDS}"  != "true" ]; then
        BUILDS="false"
        if [[ "$VERBOSE" == "true" ]]; then
            echo "${INDENT}No Builds"
        fi
        return
    fi

    j=0
    # Loop through builds and if no recent builds mark for deletion
    BUILDS="true"
    while read build_item; do
        j=$((j+1))

        BUILD=$(echo $build_item | jq -r '.number' 2>/dev/null)
        TIMESTAMP=$(echo $build_item | jq -r '.timestamp' 2>/dev/null)

        if (( TIMESTAMP > CHECK_DATE )); then
            DELETE_FOLDER="false"
            if [[ "$VERBOSE" == "true" ]]; then
                echo "${INDENT}Skipping - Build has recently run"
            fi
            break
        fi

        if (( TIMESTAMP > LAST_MOD )); then
            LAST_MOD=$TIMESTAMP
        fi
    done < <(echo $build_list | jq -c '.[]')
}

verify_syntax $@

echo "dry_run       : ${dry_run}"
echo "DAYS          : ${DAYS}"
echo "EMPTY_FOLDERS : ${EMPTY_FOLDERS}"
echo "NO_BUILDS     : ${NO_BUILDS}"
echo "VERBOSE       : ${VERBOSE}"
echo " "

# Set check date/time
CHECK_DATE=`date -d "-${DAYS} days" +%s%3N`

# Get list of top level entries in directory
dir_list=`curl -s --globoff -X POST -L -u "$JENKINS_USERNAME:$JENKINS_API_TOKEN" "${JENKINS_URL}/api/json?tree=jobs[name,url]" | jq -r .jobs`

# Process entries in list searching for directories that do not have builds that have run in X number of days
process_list "${dir_list}"

