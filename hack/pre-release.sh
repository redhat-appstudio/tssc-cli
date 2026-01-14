#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR="$(
    cd "$(dirname "$0")" >/dev/null
    pwd
)"

PROJECT_DIR="$(
    cd "$SCRIPT_DIR/.." >/dev/null
    pwd
)"

usage() {
    echo "
Usage:
    ${0##*/} [options]

Optional arguments:
    -p, --product PRODUCT
        The product on which to activate the pre-release subscription.
        Can be specified multiple times.
    -r, --tas-release-path PATH
        GitHub release path for TAS installation files.
        Example: https://github.com/securesign/releases/blob/release-1.3.1/1.3.1/stable
        Optional when using TAS product. If not provided, will auto-detect latest release
        using GitHub API (requires --github-token).
    -v, --tas-release-version VERSION
        TAS release version to use (e.g., 1.3.1).
        Defaults to \"latest\" which will fetch the most recent release.
        Requires --github-token for private repositories.
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
    ${0##*/} --product rhtas --github-token ghp_xxxxx
    
    # Specify release version (requires GitHub token)
    ${0##*/} --product rhtas --tas-release-version 1.3.1 --github-token ghp_xxxxx
    
    # Use full path (backward compatible)
    ${0##*/} --product rhtas --tas-release-path https://github.com/securesign/releases/blob/release-1.3.1/1.3.1/stable --github-token ghp_xxxxx
" >&2
}

parse_args() {
    PRODUCT_LIST=()
    TAS_RELEASE_PATH=""
    TAS_RELEASE_VERSION="latest"
    # Use GITHUB_TOKEN if set, otherwise fall back to GITOPS_GIT_TOKEN
    GITHUB_TOKEN="${GITHUB_TOKEN:-${GITOPS_GIT_TOKEN:-}}"
    while [[ $# -gt 0 ]]; do
        case $1 in
        -p|--product)
            case ${2:-} in
            developerHub|dh|rhdh)
                PRODUCT_LIST+=( "rhdh" )
                ;;
            gitops|pipelines)
                PRODUCT_LIST+=( "$2" )
                ;;
            trusted-artifact-signer|tas|rhtas)
                PRODUCT_LIST+=( "rhtas" )
                ;;
            "")
                echo "[ERROR] Product name needs to be specified after '--product'."
                usage
                exit 1
                ;;
            *)
                echo "[ERROR] Unknown product: $2"
                usage
                exit 1
                ;;
            esac
            shift
            ;;
        -r|--tas-release-path)
            if [[ -z "${2:-}" ]]; then
                echo "[ERROR] Release path needs to be specified after '--tas-release-path'."
                usage
                exit 1
            fi
            TAS_RELEASE_PATH="$2"
            shift
            ;;
        -v|--tas-release-version)
            if [[ -z "${2:-}" ]]; then
                echo "[ERROR] Release version needs to be specified after '--tas-release-version'."
                usage
                exit 1
            fi
            TAS_RELEASE_VERSION="$2"
            shift
            ;;
        -t|--github-token)
            if [[ -z "${2:-}" ]]; then
                echo "[ERROR] GitHub token needs to be specified after '--github-token'."
                usage
                exit 1
            fi
            GITHUB_TOKEN="$2"
            shift
            ;;
        -d | --debug)
            set -x
            DEBUG="--debug"
            export DEBUG
            ;;
        -h | --help)
            usage
            exit 0
            ;;
        *)
            echo "[ERROR] Unknown argument: $1"
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

h1() {
    echo "
################################################################################
# $1
################################################################################
"
}

configure_gitops(){
    # GITOPS_IIB_IMAGE="quay.io/rhtap_qe/gitops-iib:782137"

    SUBSCRIPTION="openshiftGitOps"
    CHANNEL="latest"
    SOURCE="gitops-iib"
}

configure_pipelines(){
    # PIPELINES_IMAGE="quay.io/openshift-pipeline/openshift-pipelines-pipelines-operator-bundle-container-index"
    # PIPELINES_IMAGE_TAG="v4.17-candidate"

    SUBSCRIPTION="openshiftPipelines"
    CHANNEL="latest"
    SOURCE="pipelines-iib"
}

configure_rhdh(){
    RHDH_INSTALL_SCRIPT="https://raw.githubusercontent.com/redhat-developer/rhdh-operator/main/.rhdh/scripts/install-rhdh-catalog-source.sh"
    curl -sSLO $RHDH_INSTALL_SCRIPT
    chmod +x install-rhdh-catalog-source.sh

    ./install-rhdh-catalog-source.sh --latest --install-operator rhdh

    SUBSCRIPTION="developerHub"
    CHANNEL="fast-1.9"
    SOURCE="rhdh-fast"
}

get_cluster_version() {
    # Get OpenShift cluster version (e.g., 4.20, 4.19)
    oc get clusterversion version -o jsonpath='{.status.desired.version}' | cut -d. -f1,2
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
    local curl_status
    response=$(curl "${curl_args[@]}" "$api_url" 2>&1)
    curl_status=$?
    
    if [[ $curl_status -ne 0 ]]; then
        echo "[ERROR] Failed to fetch latest release from GitHub API" >&2
        # Fallback: try listing all releases and get the first one
        echo "[INFO] Attempting fallback: fetching all releases..." >&2
        api_url="https://api.github.com/repos/${owner}/${repo}/releases"
        response=$(curl "${curl_args[@]}" "$api_url" 2>&1)
        curl_status=$?
        if [[ $curl_status -ne 0 ]]; then
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
    local curl_status
    response=$(curl "${curl_args[@]}" "$api_url" 2>&1)
    curl_status=$?
    
    if [[ $curl_status -ne 0 ]]; then
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
            echo "[ERROR] Failed to extract version from tag: $tag"
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
    
    # Convert GitHub blob URL to raw.githubusercontent.com URL
    # From: https://github.com/{owner}/{repo}/blob/{branch}/{path}
    # To:   https://raw.githubusercontent.com/{owner}/{repo}/{branch}/{path}
    local raw_url="$release_path"
    
    if [[ "$raw_url" == *"github.com"* ]] && [[ "$raw_url" == *"/blob/"* ]]; then
        # Extract owner, repo, branch, and path from blob URL
        # Example: https://github.com/securesign/releases/blob/main/1.3.1/stable
        # or: https://github.com/securesign/releases/blob/release-1.3.1/1.3.1/stable
        if [[ "$raw_url" =~ github\.com/([^/]+)/([^/]+)/blob/(.+) ]]; then
            local owner="${BASH_REMATCH[1]}"
            local repo="${BASH_REMATCH[2]}"
            local branch_and_path="${BASH_REMATCH[3]}"
            
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
            echo "[ERROR] Failed to parse GitHub URL: $release_path"
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
    
    echo "[INFO] Downloading ${file_name} from release..."
    if [[ -n "${DEBUG:-}" ]]; then
        echo "[DEBUG] URL: $raw_url"
        if [[ -n "${GITHUB_TOKEN:-}" ]]; then
            echo "[DEBUG] Using GitHub token for authentication"
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
            echo "[INFO] Download failed, trying with 'main' branch instead..."
            # Try to replace the branch with main
            # Pattern: raw.githubusercontent.com/owner/repo/branch/path -> raw.githubusercontent.com/owner/repo/refs/heads/main/path
            local fallback_url="$raw_url"
            if [[ "$fallback_url" =~ (raw\.githubusercontent\.com/[^/]+/[^/]+/)([^/]+)(/.+) ]]; then
                local base="${BASH_REMATCH[1]}"
                local path="${BASH_REMATCH[3]}"
                fallback_url="${base}refs/heads/main${path}"
                
                echo "[DEBUG] Trying fallback URL: $fallback_url"
                curl_args=(-sSLf)
                if [[ -n "${GITHUB_TOKEN:-}" ]]; then
                    curl_args+=(-H "Authorization: token ${GITHUB_TOKEN}")
                fi
                curl_args+=(-o "$output_file" "$fallback_url")
                
                if curl "${curl_args[@]}"; then
                    echo "[INFO] Successfully downloaded ${file_name} using main branch"
                    return 0
                fi
            fi
        fi
        
        echo "[ERROR] Failed to download ${file_name} from ${raw_url}"
        if [[ -z "${GITHUB_TOKEN:-}" ]] && [[ "$raw_url" == *"github.com"* ]]; then
            echo "[ERROR] This appears to be a private GitHub repository. Try using --github-token, set GITHUB_TOKEN environment variable, or set GITOPS_GIT_TOKEN as a fallback."
        fi
        return 1
    fi
    echo "[INFO] Successfully downloaded ${file_name}"
}

verify_tas_operator() {
    echo "[INFO] Verifying TAS Operator installation..."
    
    # Check Cluster Service Versions (CSV)
    echo "[INFO] Checking Cluster Service Versions (CSV)..."
    if oc get csv -n openshift-operators 2>/dev/null | grep -q "trusted-artifact-signer"; then
        CSV_NAME=$(oc get csv -n openshift-operators -o name | grep "trusted-artifact-signer" | head -n1 | cut -d/ -f2)
        CSV_PHASE=$(oc get csv "$CSV_NAME" -n openshift-operators -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
        echo "[INFO] Found TAS CSV: $CSV_NAME (Phase: $CSV_PHASE)"
        
        if [[ "$CSV_PHASE" == "Succeeded" ]]; then
            echo "[INFO] ✓ TAS Operator CSV is in Succeeded phase"
        else
            echo "[WARNING] TAS Operator CSV is in phase: $CSV_PHASE (expected: Succeeded)"
        fi
    else
        echo "[WARNING] TAS Operator CSV not found yet. It may still be installing..."
    fi
    
    # Check Pod Status
    echo "[INFO] Checking TAS Operator pods..."
    if oc get pods -n openshift-operators 2>/dev/null | grep -q "trusted-artifact-signer"; then
        echo "[INFO] TAS Operator pods:"
        oc get pods -n openshift-operators | grep "trusted-artifact-signer" || true
        
        READY_PODS=$(oc get pods -n openshift-operators -l app.kubernetes.io/instance=trusted-artifact-signer --no-headers 2>/dev/null | grep -c "Running" || echo "0")
        if [[ "$READY_PODS" -gt 0 ]]; then
            echo "[INFO] ✓ Found $READY_PODS running TAS Operator pod(s)"
        else
            echo "[WARNING] TAS Operator pods are not yet in Running state"
        fi
    else
        echo "[WARNING] TAS Operator pods not found yet. They may still be starting..."
    fi
}

configure_rhtas() {
    # Auto-detect release path if not provided
    if [[ -z "$TAS_RELEASE_PATH" ]]; then
        if [[ -z "${GITHUB_TOKEN:-}" ]]; then
            echo "[ERROR] Either --tas-release-path or --github-token is required when configuring TAS (rhtas)"
            echo "[ERROR] Option 1: Provide full path: --tas-release-path https://github.com/securesign/releases/blob/release-1.3.1/1.3.1/stable"
            echo "[ERROR] Option 2: Auto-detect latest: --github-token ghp_xxxxx"
            echo "[ERROR] Option 3: Specify version: --tas-release-version 1.3.1 --github-token ghp_xxxxx"
            exit 1
        fi
        
        echo "[INFO] Auto-detecting TAS release (version: $TAS_RELEASE_VERSION)..."
        
        local release_tag
        local version_for_path
        if [[ "$TAS_RELEASE_VERSION" == "latest" ]]; then
            release_tag=$(fetch_latest_tas_release)
            if [[ -z "$release_tag" ]]; then
                echo "[ERROR] Failed to fetch latest release tag"
                exit 1
            fi
            # Extract version from tag for path construction
            version_for_path=""
        else
            release_tag=$(fetch_tas_release_by_version "$TAS_RELEASE_VERSION")
            if [[ -z "$release_tag" ]]; then
                echo "[ERROR] Failed to fetch release tag for version: $TAS_RELEASE_VERSION"
                exit 1
            fi
            version_for_path="$TAS_RELEASE_VERSION"
        fi
        
        # Construct path from tag
        TAS_RELEASE_PATH=$(construct_tas_release_path "$release_tag" "$version_for_path")
        if [[ -z "$TAS_RELEASE_PATH" ]]; then
            echo "[ERROR] Failed to construct release path from tag: $release_tag"
            exit 1
        fi
        
        echo "[INFO] Auto-detected release path: $TAS_RELEASE_PATH"
    else
        echo "[INFO] Using provided TAS release path: $TAS_RELEASE_PATH"
    fi
    
    echo "[INFO] Configuring TAS Operator using release path: $TAS_RELEASE_PATH"
    
    # Step 1: Get cluster version for catalogSource filename
    CLUSTER_VERSION=$(get_cluster_version)
    if [[ -z "$CLUSTER_VERSION" ]]; then
        echo "[ERROR] Failed to determine cluster version"
        exit 1
    fi
    echo "[INFO] Detected OpenShift cluster version: $CLUSTER_VERSION"
    
    # Step 2: Download and apply cluster resources
    echo "[INFO] Step 2: Applying cluster resources from release..."
    
    # Download ImageDigestMirrorSet.yaml
    IMAGE_MIRROR_FILE="$SHARED_DIR/ImageDigestMirrorSet.yaml"
    if ! download_release_file "ImageDigestMirrorSet.yaml" "$TAS_RELEASE_PATH" "$IMAGE_MIRROR_FILE"; then
        echo "[ERROR] Failed to download ImageDigestMirrorSet.yaml"
        exit 1
    fi
    
    # Download catalogSource file (version-specific)
    # Convert version format: 4.19 -> 4-19 (for filename)
    VERSION_FOR_FILENAME=$(format_version_for_filename "$CLUSTER_VERSION")
    CATALOG_FILE="$SHARED_DIR/catalogSource_fbc-v${VERSION_FOR_FILENAME}.yaml"
    CATALOG_FILENAME="catalogSource_fbc-v${VERSION_FOR_FILENAME}.yaml"
    echo "[INFO] Looking for catalogSource file: $CATALOG_FILENAME"
    if ! download_release_file "$CATALOG_FILENAME" "$TAS_RELEASE_PATH" "$CATALOG_FILE"; then
        echo "[ERROR] Failed to download $CATALOG_FILENAME"
        echo "[ERROR] Please verify the cluster version ($CLUSTER_VERSION -> $VERSION_FOR_FILENAME) and that the file exists in the release"
        exit 1
    fi
    
    # Download subscription.yaml
    SUBSCRIPTION_FILE="$SHARED_DIR/subscription.yaml"
    if ! download_release_file "subscription.yaml" "$TAS_RELEASE_PATH" "$SUBSCRIPTION_FILE"; then
        echo "[ERROR] Failed to download subscription.yaml"
        exit 1
    fi
    
    # Apply ImageDigestMirrorSet
    echo "[INFO] Applying ImageDigestMirrorSet..."
    if ! oc apply -f "$IMAGE_MIRROR_FILE"; then
        echo "[ERROR] Failed to apply ImageDigestMirrorSet"
        exit 1
    fi
    echo "[INFO] ✓ ImageDigestMirrorSet applied successfully"
    
    # Apply CatalogSource
    echo "[INFO] Applying CatalogSource..."
    if ! oc apply -f "$CATALOG_FILE"; then
        echo "[ERROR] Failed to apply CatalogSource"
        exit 1
    fi
    echo "[INFO] ✓ CatalogSource applied successfully"
    
    # Apply Subscription
    echo "[INFO] Applying Subscription..."
    if ! oc apply -f "$SUBSCRIPTION_FILE"; then
        echo "[ERROR] Failed to apply Subscription"
        exit 1
    fi
    echo "[INFO] ✓ Subscription applied successfully"
    
    echo "[INFO] Operator Lifecycle Manager (OLM) is initiating TAS Operator deployment..."
    
    # Step 3: Verify Operator Installation
    echo "[INFO] Step 3: Verifying Operator Installation..."
    verify_tas_operator
    
    # Extract subscription information for configure_subscription
    # Try to get channel and source from the subscription file
    CHANNEL=$(yq '.spec.channel' "$SUBSCRIPTION_FILE" 2>/dev/null || echo "stable-v1.2")
    SOURCE=$(yq '.spec.source' "$SUBSCRIPTION_FILE" 2>/dev/null || echo "rhtas-operator")
    
    SUBSCRIPTION="openshiftTrustedArtifactSigner"
    export SUBSCRIPTION
    export CHANNEL
    export SOURCE
    
    # Update config.yaml to disable subscription management for TAS
    # This prevents Helm from trying to manage the subscription that was already created via oc apply
    config_file="$PROJECT_DIR/installer/config.yaml"
    if [[ -f "$config_file" ]]; then
        echo "[INFO] Updating config.yaml to disable TAS subscription management"
        yq -i '.tssc.products[] |= select(.name == "Trusted Artifact Signer").properties.manageSubscription = false' "$config_file"
        echo "[INFO] ✓ TAS subscription management disabled in config.yaml"
    else
        echo "[WARNING] config.yaml not found at $config_file, skipping subscription management update"
    fi
}

configure_subscription(){
    # Prepare for pre-release install capabilities
    subscription_values_file="$PROJECT_DIR/installer/charts/tssc-subscriptions/values.yaml"

    # For TAS, set source to rhtas-operator (without 's' at the end)
    if [[ "$SUBSCRIPTION" == "openshiftTrustedArtifactSigner" ]]; then
        SOURCE="rhtas-operator"
    fi

    yq -i "
        .subscriptions.$SUBSCRIPTION.channel = \"$CHANNEL\",
        .subscriptions.$SUBSCRIPTION.source = \"$SOURCE\"
    " "$subscription_values_file"
}

main() {
    parse_args "$@"
    
    # Validate TAS configuration if TAS is in product list
    if [[ " ${PRODUCT_LIST[*]} " =~ " rhtas " ]]; then
        if [[ -z "$TAS_RELEASE_PATH" ]] && [[ -z "${GITHUB_TOKEN:-}" ]]; then
            echo "[ERROR] Either --tas-release-path or --github-token is required when using TAS (rhtas) product"
            echo "[ERROR] Option 1: Provide full path: --tas-release-path <path>"
            echo "[ERROR] Option 2: Auto-detect latest: --github-token <token>"
            echo "[ERROR] Option 3: Specify version: --tas-release-version <version> --github-token <token>"
            usage
            exit 1
        fi
    fi
    
    init
    for PRODUCT in $(echo "${PRODUCT_LIST[@]}" | tr " " "\n" | sort); do
        h1 "Configuring $PRODUCT"
        "configure_$PRODUCT"
        # Only configure subscription if SUBSCRIPTION variable is set (not all products set it)
        if [[ -n "${SUBSCRIPTION:-}" ]]; then
            configure_subscription
        fi
        echo
    done
    echo "Done"
}

if [ "${BASH_SOURCE[0]}" == "$0" ]; then
    main "$@"
fi
