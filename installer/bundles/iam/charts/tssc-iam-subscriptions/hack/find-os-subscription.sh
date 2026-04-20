#!/usr/bin/env bash
#
# Finds the subscription for the given catalog name, shows the information needed
# to add the OpenShift operator subscription on this chart.
#

# OpenShift operator catalog name, for instance "openshift-pipelines-operator-rh".
declare -r CATALOG_NAME="${1:-}"

if [[ -z "${CATALOG_NAME}" ]]; then
    echo "# Usage: $$ $0 <catalog-name>"
    exit 1
fi

# Inspecting the manifests installed on the cluster, looking for the catalog name
# given in parameter. The availability of the operators depends on the OpenShift cluster
# version.
if ! oc get packagemanifests "${CATALOG_NAME}" >/dev/null; then
    echo "# ERROR: No subscription found for '${CATALOG_NAME}'!"
    exit 1
fi

# Extracing the source name, and the namespace where the catalog is available
source=$(oc get packagemanifests "${CATALOG_NAME}" \
    --output="jsonpath={.status.catalogSource}")
namespace=$(oc get packagemanifests "${CATALOG_NAME}" \
    --output="jsonpath={.status.catalogSourceNamespace}")

# Listing the channels and the startingCSVs available for the catalog.
mapfile -t channels < <(echo -en "$(oc get packagemanifests "${CATALOG_NAME}" \
    --output jsonpath='{range .status.channels[*]}{.name}\n{end}')")
mapfile -t startingCSVs < <(echo -en "$(oc get packagemanifests "${CATALOG_NAME}" \
    --output jsonpath='{range .status.channels[*]}{.currentCSV}\n{end}')")

cat <<EOS
       Name: "${CATALOG_NAME}"
     Source: "${source}"
  Namespace: "${namespace}"
    Channel: "${channels[*]}"
StartingCSV: "${startingCSVs[*]}"
EOS
