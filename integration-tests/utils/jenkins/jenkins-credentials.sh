#!/bin/bash

NAMESPACE="tssc"

REKOR_HOST="https://$(oc get routes -n tssc-tas -l "app.kubernetes.io/name=rekor-server" -o jsonpath="{.items[0].spec.host}")"
TUF_MIRROR="https://$(oc get routes -n tssc-tas -l "app.kubernetes.io/name=tuf" -o jsonpath="{.items[0].spec.host}")"

# Jenkins server details
SECRET="tssc-jenkins-integration"
JENKINS_SECRET_JSON=$(oc get secrets -n "$NAMESPACE" "$SECRET" -o json)
JENKINS__URL="$(echo "$JENKINS_SECRET_JSON" | jq -r '.data.baseUrl | @base64d')"
JENKINS__USERNAME="$(echo "$JENKINS_SECRET_JSON" | jq -r '.data.username | @base64d')"
JENKINS__TOKEN="$(echo "$JENKINS_SECRET_JSON" | jq -r '.data.token | @base64d')"


COSIGN_SECRET_JSON=$(oc get secrets -n openshift-pipelines signing-secrets -o json)
COSIGN_SECRET_KEY="$(echo "$COSIGN_SECRET_JSON" | jq -r '.data."cosign.key"')"
COSIGN_SECRET_PASSWORD="$(echo "$COSIGN_SECRET_JSON" | jq -r '.data."cosign.password"')"
COSIGN_PUBLIC_KEY="$(echo "$COSIGN_SECRET_JSON" | jq -r '.data."cosign.pub"')"

SECRET="tssc-github-integration"
GITOPS_SECRET_JSON=$(oc get secrets -n "$NAMESPACE" "$SECRET" -o json)
GITOPS_GIT_TOKEN="$(echo "$GITOPS_SECRET_JSON" | jq -r '.data.token | @base64d')"
GITOPS_AUTH_USERNAME="$(echo "$GITOPS_SECRET_JSON" | jq -r '.data.username | @base64d')"


SECRET="tssc-quay-integration"
REGISTRY_SECRET_JSON=$(oc get secrets -n "$NAMESPACE" "$SECRET" -o json)
QUAY_USERNAME="$(
    echo "$REGISTRY_SECRET_JSON" \
    | jq -r '.data.".dockerconfigjson" | @base64d' \
    | jq -r '.auths | to_entries[0].value.auth | @base64d' \
    | cut -d: -f1
)"
QUAY_PASSWORD="$(
    echo "$REGISTRY_SECRET_JSON" \
    | jq -r '.data.".dockerconfigjson" | @base64d' \
    | jq -r '.auths | to_entries[0].value.auth | @base64d' \
    | cut -d: -f2-
)"

SECRET="tssc-acs-integration"
ACS_ENDPOINT="$(oc get secrets -n "$NAMESPACE" "$SECRET" -o json | jq -r '.data.endpoint | @base64d')"
ACS_TOKEN="$(oc get secrets -n "$NAMESPACE" "$SECRET" -o json | jq -r '.data.token | @base64d')"

# SBOM automatic upload creds
SECRET="tssc-trustification-integration"
TRUSTIFICATION_BOMBASTIC_API_URL="$(oc get secret -n "$NAMESPACE" "$SECRET" --template={{.data.bombastic_api_url}} | base64 -d)"
TRUSTIFICATION_OIDC_ISSUER_URL="$(oc get secret -n "$NAMESPACE" "$SECRET" --template={{.data.oidc_issuer_url}} | base64 -d)"
TRUSTIFICATION_OIDC_CLIENT_ID="$(oc get secret -n "$NAMESPACE" "$SECRET" --template={{.data.oidc_client_id}} | base64 -d)"
TRUSTIFICATION_OIDC_CLIENT_SECRET="$(oc get secret -n "$NAMESPACE" "$SECRET" --template={{.data.oidc_client_secret}} | base64 -d)"
TRUSTIFICATION_SUPPORTED_CYCLONEDX_VERSION="$(oc get secret -n "$NAMESPACE" "$SECRET" --template={{.data.supported_cyclonedx_version}} | base64 -d)"


# Arrays of credential details
CREDENTIAL_IDS=("TUF_MIRROR" "REKOR_HOST" "IMAGE_REGISTRY_USER" "IMAGE_REGISTRY_PASSWORD" "ROX_API_TOKEN" "ROX_CENTRAL_ENDPOINT" "GITOPS_AUTH_USERNAME" "GITOPS_AUTH_PASSWORD" "COSIGN_SECRET_PASSWORD" "COSIGN_SECRET_KEY" "COSIGN_PUBLIC_KEY" "TRUSTIFICATION_BOMBASTIC_API_URL" "TRUSTIFICATION_OIDC_ISSUER_URL" "TRUSTIFICATION_OIDC_CLIENT_ID" "TRUSTIFICATION_OIDC_CLIENT_SECRET" "TRUSTIFICATION_SUPPORTED_CYCLONEDX_VERSION")
SECRETS=($TUF_MIRROR $REKOR_HOST $QUAY_USERNAME $QUAY_PASSWORD $ACS_TOKEN $ACS_ENDPOINT $GITOPS_AUTH_USERNAME $GITOPS_GIT_TOKEN $COSIGN_SECRET_PASSWORD $COSIGN_SECRET_KEY $COSIGN_PUBLIC_KEY $TRUSTIFICATION_BOMBASTIC_API_URL $TRUSTIFICATION_OIDC_ISSUER_URL $TRUSTIFICATION_OIDC_CLIENT_ID $TRUSTIFICATION_OIDC_CLIENT_SECRET $TRUSTIFICATION_SUPPORTED_CYCLONEDX_VERSION)

# Function to add a single credential
add_secret() {
    local id=$1
    local secret=$2

    local json=$(cat <<EOF
{
  "": "0",
  "credentials": {
    "scope": "GLOBAL",
    "id": "${id}",
    "secret": "${secret}",
    "description": "",
    "\$class": "org.jenkinsci.plugins.plaincredentials.impl.StringCredentialsImpl"
  }
}
EOF
)

    create_credentials "$json"    
}

add_username_with_password() {
    local id=$1
    local username=$2
    local password=$3

    local json=$(cat <<EOF
{
  "": "0",
  "credentials": {
    "scope": "GLOBAL",
    "id": "${id}",
    "username": "${username}",
    "password": "${password}",
    "description": "",
    "\$class": "com.cloudbees.plugins.credentials.impl.UsernamePasswordCredentialsImpl"
  }
}
EOF
)

    create_credentials "$json"
}

create_credentials() {
    local json=$1

    curl -X POST "$JENKINS__URL/credentials/store/system/domain/_/createCredentials" \
    --user "$JENKINS__USERNAME:$JENKINS__TOKEN" \
    --data-urlencode "json=$json"
}


# Add multiple credentials
for i in "${!CREDENTIAL_IDS[@]}"; do
    add_secret "${CREDENTIAL_IDS[$i]}" "${SECRETS[$i]}"
    echo "Credential ${CREDENTIAL_IDS[$i]} is set" 
done

# Add usernames with passwords
add_username_with_password "QUAY_IO_CREDS" $QUAY_USERNAME $QUAY_PASSWORD
add_username_with_password "GITOPS_CREDENTIALS" $GITOPS_AUTH_USERNAME $GITOPS_GIT_TOKEN
