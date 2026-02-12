#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Source common functions
SCRIPT_DIR="$(
    cd "$(dirname "$0")" >/dev/null || exit
    pwd
)"
# shellcheck source=hack/pre-release/pre-release-common.sh
source "$SCRIPT_DIR/pre-release-common.sh"

usage() {
    echo "
Usage:
    ${0##*/} [options]

Optional arguments:
    -r, --release-path PATH
        GitHub release path for TAS installation files.
        Example: https://github.com/securesign/releases/blob/release-1.3.1/1.3.1/stable
        Optional. If not provided, will auto-detect latest release using GitHub API (requires --github-token).
    -v, --release-version VERSION
        TAS release version to use (e.g., 1.3.1).
        Defaults to \"latest\" which will fetch the most recent release.
        Requires --github-token for private repositories.
    -o, --operator-version VERSION
        Exact TAS operator CSV version to install (e.g., rhtas-operator.v1.3.2).
        If specified, this will override the startingCSV in the subscription.yaml.
        Optional. If not provided, uses the startingCSV from the release subscription.yaml.
    -t, --github-token TOKEN
        GitHub personal access token for accessing private repositories.
        Required for auto-detecting latest TAS release or accessing private repos.
        Can also be set via GITHUB_TOKEN environment variable.
    -d, --debug
        Activate tracing/debug mode.
    -h, --help
        Display this message.

Examples:
    # Auto-detect latest release (requires GitHub token)
    ${0##*/} --github-token ghp_xxxxx
    
    # Specify release version (requires GitHub token)
    ${0##*/} --release-version 1.3.1 --github-token ghp_xxxxx
    
    # Specify exact operator version
    ${0##*/} --release-version 1.3.2 --operator-version rhtas-operator.v1.3.2 --github-token ghp_xxxxx
    
    # Use full path (backward compatible)
    ${0##*/} --release-path https://github.com/securesign/releases/blob/release-1.3.1/1.3.1/stable --github-token ghp_xxxxx
" >&2
}

parse_args() {
    TAS_RELEASE_PATH=""
    TAS_RELEASE_VERSION="latest"
    TAS_OPERATOR_VERSION=""
    init_github_token
    while [[ $# -gt 0 ]]; do
        case $1 in
        -r|--release-path)
            if [[ -z "${2:-}" ]]; then
                echo "[ERROR] Release path needs to be specified after '--release-path'." >&2
                usage
                exit 1
            fi
            TAS_RELEASE_PATH="$2"
            shift
            ;;
        -v|--release-version)
            if [[ -z "${2:-}" ]]; then
                echo "[ERROR] Release version needs to be specified after '--release-version'." >&2
                usage
                exit 1
            fi
            TAS_RELEASE_VERSION="$2"
            shift
            ;;
        -o|--operator-version)
            if [[ -z "${2:-}" ]]; then
                echo "[ERROR] Operator version needs to be specified after '--operator-version'." >&2
                usage
                exit 1
            fi
            TAS_OPERATOR_VERSION="$2"
            shift
            ;;
        -t|--github-token)
            if [[ -z "${2:-}" ]]; then
                echo "[ERROR] GitHub token needs to be specified after '--github-token'." >&2
                usage
                exit 1
            fi
            GITHUB_TOKEN="$2"
            shift
            ;;
        -d|--debug)
            set -x
            DEBUG="--debug"
            export DEBUG
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "[ERROR] Unknown argument: $1" >&2
            usage
            exit 1
            ;;
        esac
        shift
    done
}

init() {
    SHARED_DIR="$(mktemp -d)"
    cd "$SHARED_DIR"
    export SHARED_DIR
    trap cleanup EXIT
}

cleanup() {
    rm -rf "$SHARED_DIR"
}

format_version_for_filename() {
    # Convert version format from 4.19 to 4-19 (for catalogSource filename)
    local version=$1
    echo "${version//./-}"
}

# Fetch latest release tag from GitHub API
fetch_latest_tas_release() {
    local owner="securesign"
    local repo="releases"
    local api_url="https://api.github.com/repos/${owner}/${repo}/releases/latest"
    
    if [[ -z "${GITHUB_TOKEN:-}" ]]; then
        echo "[ERROR] GitHub token is required to fetch latest release from private repository" >&2
        echo "[ERROR] Please provide --github-token or set GITHUB_TOKEN environment variable" >&2
        return 1
    fi
    
    echo "[INFO] Fetching latest TAS release from GitHub API..." >&2
    if [[ -n "${DEBUG:-}" ]]; then
        echo "[DEBUG] API URL: $api_url" >&2
    fi
    
    local curl_args=(-sSLf)
    curl_args+=(-H "Authorization: token ${GITHUB_TOKEN}")
    curl_args+=(-H "Accept: application/vnd.github.v3+json")
    
    local response
    if ! response=$(curl "${curl_args[@]}" "$api_url" 2>&1); then
        echo "[ERROR] Failed to fetch latest release from GitHub API" >&2
        # Fallback: try listing all releases and get the first one
        echo "[INFO] Attempting fallback: fetching all releases..." >&2
        api_url="https://api.github.com/repos/${owner}/${repo}/releases"
        if ! response=$(curl "${curl_args[@]}" "$api_url" 2>&1); then
            echo "[ERROR] Failed to fetch releases from GitHub API" >&2
            return 1
        fi
        # Extract the first release tag_name
        local latest_tag
        latest_tag=$(echo "$response" | grep -o '"tag_name":"[^"]*"' | head -n1 | cut -d'"' -f4)
        if [[ -z "$latest_tag" ]]; then
            echo "[ERROR] Failed to parse latest release tag from GitHub API response" >&2
            if [[ -n "${DEBUG:-}" ]]; then
                echo "[DEBUG] API Response: $response" >&2
            fi
            return 1
        fi
        echo "[INFO] Found latest release tag: $latest_tag" >&2
        echo "$latest_tag"
        return 0
    fi
    
    # Extract the tag_name from the latest release response
    local latest_tag
    latest_tag=$(echo "$response" | grep -o '"tag_name":"[^"]*"' | head -n1 | cut -d'"' -f4)
    
    if [[ -z "$latest_tag" ]]; then
        echo "[ERROR] Failed to parse latest release tag from GitHub API response" >&2
        if [[ -n "${DEBUG:-}" ]]; then
            echo "[DEBUG] API Response: $response" >&2
        fi
        return 1
    fi
    
    echo "[INFO] Found latest release tag: $latest_tag" >&2
    echo "$latest_tag"
}

# Fetch specific release tag from GitHub API by version
fetch_tas_release_by_version() {
    local version=$1
    local owner="securesign"
    local repo="releases"
    local api_url="https://api.github.com/repos/${owner}/${repo}/releases"
    
    if [[ -z "${GITHUB_TOKEN:-}" ]]; then
        echo "[ERROR] GitHub token is required to fetch release from private repository" >&2
        echo "[ERROR] Please provide --github-token or set GITHUB_TOKEN environment variable" >&2
        return 1
    fi
    
    echo "[INFO] Fetching TAS release version $version from GitHub API..." >&2
    if [[ -n "${DEBUG:-}" ]]; then
        echo "[DEBUG] API URL: $api_url" >&2
        echo "[DEBUG] Looking for version: $version" >&2
    fi
    
    local curl_args=(-sSLf)
    curl_args+=(-H "Authorization: token ${GITHUB_TOKEN}")
    curl_args+=(-H "Accept: application/vnd.github.v3+json")
    
    local response
    if ! response=$(curl "${curl_args[@]}" "$api_url" 2>&1); then
        echo "[ERROR] Failed to fetch releases from GitHub API" >&2
        return 1
    fi
    
    # Try to find a release matching the version
    # Look for tag_name containing the version (e.g., "release-1.3.1" or "v1.3.1")
    local matching_tag
    matching_tag=$(echo "$response" | grep -o '"tag_name":"[^"]*' | grep -i "$version" | head -n1 | cut -d'"' -f4)
    
    if [[ -z "$matching_tag" ]]; then
        echo "[ERROR] No release found matching version: $version" >&2
        echo "[ERROR] Available releases:" >&2
        echo "$response" | grep -o '"tag_name":"[^"]*"' | cut -d'"' -f4 | head -n5 | sed 's/^/  - /' >&2
        return 1
    fi
    
    echo "[INFO] Found release tag: $matching_tag" >&2
    echo "$matching_tag"
}

# Construct TAS release path from tag
construct_tas_release_path() {
    local tag=$1
    local version=$2
    
    # Extract version from tag if not provided
    if [[ -z "$version" ]]; then
        # Try to extract version from tag (e.g., "release-1.3.1" -> "1.3.1")
        if [[ "$tag" =~ release-([0-9]+\.[0-9]+\.[0-9]+) ]]; then
            version="${BASH_REMATCH[1]}"
        elif [[ "$tag" =~ v?([0-9]+\.[0-9]+\.[0-9]+) ]]; then
            version="${BASH_REMATCH[1]}"
        else
            echo "[ERROR] Failed to extract version from tag: $tag" >&2
            return 1
        fi
    fi
    
    # Construct the path: https://github.com/securesign/releases/blob/{tag}/{version}/stable
    local path="https://github.com/securesign/releases/blob/${tag}/${version}/stable"
    echo "$path"
}

download_release_file() {
    local file_name=$1
    local release_path=$2
    local output_file=$3
    
    # Convert GitHub blob/tree URL to raw.githubusercontent.com URL
    # From: https://github.com/{owner}/{repo}/blob/{branch}/{path}
    #   or: https://github.com/{owner}/{repo}/tree/{branch}/{path}
    # To:   https://raw.githubusercontent.com/{owner}/{repo}/{branch}/{path}
    local raw_url="$release_path"
    
      if [[ "$raw_url" == *"github.com"* ]] && { [[ "$raw_url" == *"/blob/"* ]] || [[ "$raw_url" == *"/tree/"* ]]; }; then
        # Extract owner, repo, branch, and path from blob/tree URL
        # Example: https://github.com/securesign/releases/blob/main/1.3.1/stable
        # or: https://github.com/securesign/releases/blob/release-1.3.1/1.3.1/stable
        # or: https://github.com/securesign/releases/tree/release-1.3.2/1.3.2/stable
        if [[ "$raw_url" =~ github\.com/([^/]+)/([^/]+)/(blob|tree)/(.+) ]]; then
            local owner="${BASH_REMATCH[1]}"
            local repo="${BASH_REMATCH[2]}"
            local branch_and_path="${BASH_REMATCH[4]}"
            
            # Check if branch is "main" and convert to refs/heads/main format
            # GitHub raw URLs use refs/heads/main instead of just main
            # Example: blob/main/1.3.1/stable -> refs/heads/main/1.3.1/stable
            if [[ "$branch_and_path" == main/* ]]; then
                branch_and_path="refs/heads/${branch_and_path}"
            elif [[ "$branch_and_path" == main ]]; then
                branch_and_path="refs/heads/main"
            fi
            
            raw_url="https://raw.githubusercontent.com/${owner}/${repo}/${branch_and_path}"
        else
            echo "[ERROR] Failed to parse GitHub URL: $release_path" >&2
            return 1
        fi
    elif [[ "$raw_url" == *"raw.githubusercontent.com"* ]]; then
        # Already a raw URL, use as-is
        :
    else
        # Assume it's already a raw URL or different format, use as-is
        :
    fi
    
    # Ensure we have the testingResources path
    if [[ "$raw_url" != *"/testingResources/"* ]]; then
        raw_url="${raw_url%/}/testingResources/${file_name}"
    else
        raw_url="${raw_url%/}/${file_name}"
    fi
    
    echo "[INFO] Downloading ${file_name} from release..." >&2
    if [[ -n "${DEBUG:-}" ]]; then
        echo "[DEBUG] URL: $raw_url" >&2
        if [[ -n "${GITHUB_TOKEN:-}" ]]; then
            echo "[DEBUG] Using GitHub token for authentication" >&2
        fi
    fi
    
    # Build curl command with optional GitHub token authentication
    local curl_args=(-sSLf)
    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
        curl_args+=(-H "Authorization: token ${GITHUB_TOKEN}")
    fi
    curl_args+=(-o "$output_file" "$raw_url")
    
    if ! curl "${curl_args[@]}"; then
        # If download failed and URL contains a branch that's not main, try with main branch
        if [[ "$raw_url" == *"raw.githubusercontent.com"* ]] && [[ "$raw_url" != *"refs/heads/main"* ]] && [[ "$raw_url" != *"/main/"* ]]; then
            echo "[INFO] Download failed, trying with 'main' branch instead..." >&2
            # Try to replace the branch with main
            # Pattern: raw.githubusercontent.com/owner/repo/branch/path -> raw.githubusercontent.com/owner/repo/refs/heads/main/path
            local fallback_url="$raw_url"
            if [[ "$fallback_url" =~ (raw\.githubusercontent\.com/[^/]+/[^/]+/)([^/]+)(/.+) ]]; then
                local base="${BASH_REMATCH[1]}"
                local path="${BASH_REMATCH[3]}"
                fallback_url="${base}refs/heads/main${path}"
                
                echo "[DEBUG] Trying fallback URL: $fallback_url" >&2
                curl_args=(-sSLf)
                if [[ -n "${GITHUB_TOKEN:-}" ]]; then
                    curl_args+=(-H "Authorization: token ${GITHUB_TOKEN}")
                fi
                curl_args+=(-o "$output_file" "$fallback_url")
                
                if curl "${curl_args[@]}"; then
                    echo "[INFO] Successfully downloaded ${file_name} using main branch" >&2
                    return 0
                fi
            fi
        fi
        
        echo "[ERROR] Failed to download ${file_name} from ${raw_url}" >&2
        if [[ -z "${GITHUB_TOKEN:-}" ]] && [[ "$raw_url" == *"github.com"* ]]; then
            echo "[ERROR] This appears to be a private GitHub repository. Try using --github-token, set GITHUB_TOKEN environment variable, or set GITOPS_GIT_TOKEN as a fallback." >&2
        fi
        return 1
    fi
    echo "[INFO] Successfully downloaded ${file_name}" >&2
}

configure_tas() {
    # Auto-detect release path if not provided
    if [[ -z "$TAS_RELEASE_PATH" ]]; then
        if [[ -z "${GITHUB_TOKEN:-}" ]]; then
            echo "[ERROR] Either --release-path or --github-token is required when configuring TAS" >&2
            echo "[ERROR] Option 1: Provide full path: --release-path https://github.com/securesign/releases/blob/release-1.3.1/1.3.1/stable" >&2
            echo "[ERROR] Option 2: Auto-detect latest: --github-token ghp_xxxxx" >&2
            echo "[ERROR] Option 3: Specify version: --release-version 1.3.1 --github-token ghp_xxxxx" >&2
            exit 1
        fi
        
        echo "[INFO] Auto-detecting TAS release (version: $TAS_RELEASE_VERSION)..." >&2
        
        local release_tag
        local version_for_path
        if [[ "$TAS_RELEASE_VERSION" == "latest" ]]; then
            release_tag=$(fetch_latest_tas_release)
            if [[ -z "$release_tag" ]]; then
                echo "[ERROR] Failed to fetch latest release tag" >&2
                exit 1
            fi
            # Extract version from tag for path construction
            version_for_path=""
        else
            release_tag=$(fetch_tas_release_by_version "$TAS_RELEASE_VERSION")
            if [[ -z "$release_tag" ]]; then
                echo "[ERROR] Failed to fetch release tag for version: $TAS_RELEASE_VERSION" >&2
                exit 1
            fi
            version_for_path="$TAS_RELEASE_VERSION"
        fi
        
        # Construct path from tag
        TAS_RELEASE_PATH=$(construct_tas_release_path "$release_tag" "$version_for_path")
        if [[ -z "$TAS_RELEASE_PATH" ]]; then
            echo "[ERROR] Failed to construct release path from tag: $release_tag" >&2
            exit 1
        fi
        
        echo "[INFO] Auto-detected release path: $TAS_RELEASE_PATH" >&2
    else
        echo "[INFO] Using provided TAS release path: $TAS_RELEASE_PATH" >&2
    fi
    
    echo "[INFO] Configuring TAS Operator using release path: $TAS_RELEASE_PATH" >&2
    
    # Step 1: Get cluster version for catalogSource filename
    CLUSTER_VERSION=$(get_cluster_version)
    if [[ -z "$CLUSTER_VERSION" ]]; then
        echo "[ERROR] Failed to determine cluster version" >&2
        exit 1
    fi
    echo "[INFO] Detected OpenShift cluster version: $CLUSTER_VERSION" >&2
    
    # Step 2: Download and apply cluster resources
    echo "[INFO] Step 2: Applying cluster resources from release..." >&2
    
    # Download ImageDigestMirrorSet.yaml
    IMAGE_MIRROR_FILE="$SHARED_DIR/ImageDigestMirrorSet.yaml"
    if ! download_release_file "ImageDigestMirrorSet.yaml" "$TAS_RELEASE_PATH" "$IMAGE_MIRROR_FILE"; then
        echo "[ERROR] Failed to download ImageDigestMirrorSet.yaml" >&2
        exit 1
    fi
    
    # Download catalogSource file (version-specific)
    # Convert version format: 4.19 -> 4-19 (for filename)
    VERSION_FOR_FILENAME=$(format_version_for_filename "$CLUSTER_VERSION")
    CATALOG_FILE="$SHARED_DIR/catalogSource_fbc-v${VERSION_FOR_FILENAME}.yaml"
    CATALOG_FILENAME="catalogSource_fbc-v${VERSION_FOR_FILENAME}.yaml"
    echo "[INFO] Looking for catalogSource file: $CATALOG_FILENAME" >&2
    if ! download_release_file "$CATALOG_FILENAME" "$TAS_RELEASE_PATH" "$CATALOG_FILE"; then
        echo "[ERROR] Failed to download $CATALOG_FILENAME" >&2
        echo "[ERROR] Please verify the cluster version ($CLUSTER_VERSION -> $VERSION_FOR_FILENAME) and that the file exists in the release" >&2
        exit 1
    fi
    
    # Download subscription.yaml
    SUBSCRIPTION_FILE="$SHARED_DIR/subscription.yaml"
    if ! download_release_file "subscription.yaml" "$TAS_RELEASE_PATH" "$SUBSCRIPTION_FILE"; then
        echo "[ERROR] Failed to download subscription.yaml" >&2
        exit 1
    fi
    
    # Apply ImageDigestMirrorSet
    echo "[INFO] Applying ImageDigestMirrorSet..." >&2
    if ! oc apply -f "$IMAGE_MIRROR_FILE"; then
        echo "[ERROR] Failed to apply ImageDigestMirrorSet" >&2
        exit 1
    fi
    echo "[INFO] ✓ ImageDigestMirrorSet applied successfully" >&2
    
    # Apply CatalogSource
    echo "[INFO] Applying CatalogSource..." >&2
    if ! oc apply -f "$CATALOG_FILE"; then
        echo "[ERROR] Failed to apply CatalogSource" >&2
        exit 1
    fi
    echo "[INFO] ✓ CatalogSource applied successfully" >&2
    
    # Get the catalog source name from the CatalogSource file
    CATALOG_SOURCE_NAME=$(yq '.metadata.name' "$CATALOG_FILE" 2>/dev/null || echo "")
    if [[ -z "$CATALOG_SOURCE_NAME" ]]; then
        echo "[ERROR] Could not determine catalog source name from $CATALOG_FILE" >&2
        exit 1
    fi
    echo "[INFO] Pre-release catalog source name: $CATALOG_SOURCE_NAME" >&2
    
    # Apply Subscription
    # Ensure the subscription has the correct name and namespace that the installer expects
    # The installer expects: name: rhtas-operator, namespace: openshift-operators
    echo "[INFO] Ensuring subscription matches installer expectations..." >&2
    SUBSCRIPTION_NAME=$(yq '.metadata.name' "$SUBSCRIPTION_FILE" 2>/dev/null || echo "")
    SUBSCRIPTION_NAMESPACE=$(yq '.metadata.namespace' "$SUBSCRIPTION_FILE" 2>/dev/null || echo "")
    
    # Update subscription to match installer expectations if needed
    if [[ "$SUBSCRIPTION_NAME" != "rhtas-operator" ]] || [[ "$SUBSCRIPTION_NAMESPACE" != "openshift-operators" ]]; then
        echo "[INFO] Updating subscription name/namespace to match installer expectations..." >&2
        yq -i '.metadata.name = "rhtas-operator"' "$SUBSCRIPTION_FILE"
        yq -i '.metadata.namespace = "openshift-operators"' "$SUBSCRIPTION_FILE"
        echo "[INFO] Updated subscription: name=rhtas-operator, namespace=openshift-operators" >&2
    fi
    
    # CRITICAL: Ensure subscription uses the pre-release catalog source, not the default "rhtas-operator" or "redhat-operators"
    CURRENT_SUB_SOURCE=$(yq '.spec.source' "$SUBSCRIPTION_FILE" 2>/dev/null || echo "")
    if [[ "$CURRENT_SUB_SOURCE" == "rhtas-operator" ]] || [[ "$CURRENT_SUB_SOURCE" == "redhat-operators" ]]; then
        echo "[INFO] Updating subscription source from '$CURRENT_SUB_SOURCE' to pre-release catalog source '$CATALOG_SOURCE_NAME'..." >&2
        yq -i ".spec.source = \"$CATALOG_SOURCE_NAME\"" "$SUBSCRIPTION_FILE"
        echo "[INFO] ✓ Subscription source updated to use pre-release catalog source" >&2
    elif [[ "$CURRENT_SUB_SOURCE" != "$CATALOG_SOURCE_NAME" ]]; then
        echo "[WARNING] Subscription source ($CURRENT_SUB_SOURCE) does not match catalog source name ($CATALOG_SOURCE_NAME)" >&2
        echo "[INFO] Updating subscription source to use pre-release catalog source..." >&2
        yq -i ".spec.source = \"$CATALOG_SOURCE_NAME\"" "$SUBSCRIPTION_FILE"
        echo "[INFO] ✓ Subscription source updated to: $CATALOG_SOURCE_NAME" >&2
    else
        echo "[INFO] ✓ Subscription is already using correct catalog source: $CATALOG_SOURCE_NAME" >&2
    fi
    
    # Override startingCSV if operator version is specified
    if [[ -n "${TAS_OPERATOR_VERSION:-}" ]]; then
        echo "[INFO] Setting operator version to: $TAS_OPERATOR_VERSION" >&2
        yq -i ".spec.startingCSV = \"$TAS_OPERATOR_VERSION\"" "$SUBSCRIPTION_FILE"
        echo "[INFO] Updated subscription startingCSV to: $TAS_OPERATOR_VERSION" >&2
    fi
    
    echo "[INFO] Applying Subscription..." >&2
    if ! oc apply -f "$SUBSCRIPTION_FILE"; then
        echo "[ERROR] Failed to apply Subscription" >&2
        exit 1
    fi
    echo "[INFO] ✓ Subscription applied successfully" >&2
    
    echo "[INFO] Subscription applied - OLM will install the TAS Operator asynchronously" >&2
    
    # Extract subscription information for config updates
    # Try to get channel and source from the subscription file
    CHANNEL=$(yq '.spec.channel' "$SUBSCRIPTION_FILE" 2>/dev/null || echo "stable-v1.3")
    SOURCE=$(yq '.spec.source' "$SUBSCRIPTION_FILE" 2>/dev/null || echo "rhtas-operator")
    
    # Update config.yaml to disable subscription management for TAS
    # This prevents Helm from trying to manage the subscription that was already created via oc apply
    config_file="$PROJECT_DIR/installer/config.yaml"
    if [[ -f "$config_file" ]]; then
        echo "[INFO] Updating config.yaml to disable TAS subscription management" >&2
        yq -i '.tssc.products[] |= (select(.name == "Trusted Artifact Signer") | .properties.manageSubscription = false)' "$config_file"
        echo "[INFO] ✓ TAS subscription management disabled in config.yaml" >&2
    else
        echo "[WARNING] config.yaml not found at $config_file, skipping subscription management update" >&2
    fi
    
    # Update values.yaml to use the correct source and channel from the pre-release subscription
    # This ensures the installer references match what was actually installed
    values_file="$PROJECT_DIR/installer/charts/tssc-subscriptions/values.yaml"
    if [[ -f "$values_file" ]]; then
        echo "[INFO] Updating values.yaml to use pre-release TAS catalog source and channel" >&2
        echo "[INFO] Setting source to: $SOURCE, channel to: $CHANNEL" >&2
        yq -i ".subscriptions.openshiftTrustedArtifactSigner.source = \"$SOURCE\"" "$values_file"
        yq -i ".subscriptions.openshiftTrustedArtifactSigner.channel = \"$CHANNEL\"" "$values_file"
        echo "[INFO] ✓ TAS subscription source and channel updated in values.yaml" >&2
    else
        echo "[WARNING] values.yaml not found at $values_file, skipping subscription source/channel update" >&2
    fi
}

main() {
    parse_args "$@"
    
    # Validate TAS configuration
    if [[ -z "$TAS_RELEASE_PATH" ]] && [[ -z "${GITHUB_TOKEN:-}" ]]; then
        echo "[ERROR] Either --release-path or --github-token is required when configuring TAS" >&2
        echo "[ERROR] Option 1: Provide full path: --release-path <path>" >&2
        echo "[ERROR] Option 2: Auto-detect latest: --github-token <token>" >&2
        echo "[ERROR] Option 3: Specify version: --release-version <version> --github-token <token>" >&2
        usage
        exit 1
    fi
    
    init
    configure_tas
    echo "Done" >&2
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
fi
