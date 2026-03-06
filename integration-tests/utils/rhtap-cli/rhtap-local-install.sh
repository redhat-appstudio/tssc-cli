#!/usr/bin/env bash

#quit if exit status of any cmd is a non-zero value
set -o errexit
set -o nounset
set -o pipefail

export acs_install_enabled="false"
export quay_install_enabled="false"
export github_enabled="true"
export gitlab_enabled="true"
export jenkins_enabled="true"
export bitbucket_enabled="false"

export DEVELOPER_HUB__CATALOG__URL="https://github.com/redhat-appstudio/tssc-sample-templates/blob/main/all.yaml" 
# Variables for GitHub integration
export GITHUB__APP__ID="<REPLACE_ME>"
export GITHUB__APP__CLIENT__ID="<REPLACE_ME>"
export GITHUB__APP__CLIENT__SECRET="<REPLACE_ME>"
###"/Users/xinjiang/Downloads/rhtap-test.2024-10-17.private-key.pem"
GITHUB_APP_PRIVATE_KEY_BASE64="<REPLACE_ME, for example, you can get the value with the command $(cat private-key.pem | base64 -w 0)>"
export GITHUB__APP__PRIVATE_KEY="$(echo -n ${GITHUB_APP_PRIVATE_KEY_BASE64} | base64 -d)"
export GITOPS__GIT_TOKEN="<REPLACE_ME>"
export GITHUB__APP__WEBHOOK__SECRET="<REPLACE_ME, any random string>"
# Variables for Gitlab integration
export GITLAB__TOKEN="<REPLACE_ME>"
# Variables for Jenkins integration
export JENKINS_API_TOKEN="<REPLACE_ME>"
export JENKINS_URL="<REPLACE_ME>"
export JENKINS_USERNAME="<REPLACE_ME>"
# Variables for quay.io integration
export QUAY__DOCKERCONFIGJSON="<REPLACE_ME>"
export QUAY__API_TOKEN="<REPLACE_ME>"
# Variables for ACS integration
export ACS__CENTRAL_ENDPOINT="<REPLACE_ME>"
export ACS__API_TOKEN="<REPLACE_ME>"
## variables for Bitbucket integration
export BITBUCKET_USERNAME="<REPLACE_ME>"
export BITBUCKET_APP_PASSWORD="<REPLACE_ME>"

jwt_token() {
  app_id=$1     # App ID as first argument
  pem=$2        # content of the private key as second argument

  now=$(date +%s)
  iat=$((now - 60))  # Issues 60 seconds in the past
  exp=$((now + 600)) # Expires 10 minutes in the future

  b64enc() { openssl base64 | tr -d '=' | tr '/+' '_-' | tr -d '\n'; }

  header_json='{
      "typ":"JWT",
      "alg":"RS256"
  }'
  # Header encode
  header=$(echo -n "${header_json}" | b64enc)

  payload_json='{
      "iat":'"${iat}"',
      "exp":'"${exp}"',
      "iss":'"${app_id}"'
  }'
  # Payload encode
  payload=$(echo -n "${payload_json}" | b64enc)

  # Signature
  header_payload="${header}"."${payload}"
  signature=$(
      openssl dgst -sha256 -sign <(echo -n "${pem}") \
          <(echo -n "${header_payload}") | b64enc
  )

  # Create JWT
  JWT_TOKEN="${header_payload}"."${signature}"
}

update_github_app() {
  echo "[INFO]Update GitHub App"
  webhook_url=https://$(kubectl -n openshift-pipelines get route pipelines-as-code-controller -o 'jsonpath={.spec.host}')
  # github_private_key="$(echo -n ${GITHUB__APP__PRIVATE_KEY} | base64 -d)"
  jwt_token "$GITHUB__APP__ID" "$GITHUB__APP__PRIVATE_KEY"
  curl \
    -X PATCH \
    -H "Accept: application/vnd.github.v3+json" \
    -H "Authorization: Bearer $JWT_TOKEN" \
    https://api.github.com/app/hook/config \
    -d "{\"content_type\":\"json\",\"insecure_ssl\":\"0\",\"secret\":\"$GITHUB__APP__WEBHOOK__SECRET\",\"url\":\"$webhook_url\"}" &>/dev/null
}

run-rhtap-e2e() {
  export GITLAB_TOKEN="$GITLAB__TOKEN"
  export GITHUB_TOKEN="$GITOPS__GIT_TOKEN"

  export APPLICATION_ROOT_NAMESPACE="rhtap-app"
  export GITHUB_ORGANIZATION="<REPLACE_ME>"
  export GITLAB_ORGANIZATION="<REPLACE_ME>"
  export QUAY_IMAGE_ORG="<REPLACE_ME>"
  export IMAGE_REGISTRY="<REPLACE_ME>" ## "quay.io" or "$(kubectl -n rhtap-quay get route rhtap-quay-quay -o 'jsonpath={.spec.host}')"

  export RED_HAT_DEVELOPER_HUB_URL="https://$(kubectl -n rhtap get route backstage-developer-hub -o 'jsonpath={.spec.host}')"
  export NODE_TLS_REJECT_UNAUTHORIZED=0

  echo "[INFO] Clone rhtap-e2e repo"
  if [ -d rhtap-e2e ]; then
    rm -rf rhtap-e2e
  fi
  git clone https://github.com/redhat-appstudio/rhtap-e2e.git
  cd rhtap-e2e

  yarn && yarn test tests/gpts/github/quarkus.tekton.test.ts
  # yarn && yarn test runTestsByPath tests/gpts/github/
}

bash integration-tests/scripts/install.sh
update_github_app
# run-rhtap-e2e