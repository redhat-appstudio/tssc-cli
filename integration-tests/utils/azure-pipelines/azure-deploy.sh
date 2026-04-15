#!/bin/bash
set -e

# Create new project 'azure-build'
oc new-project azure-build
# Create ConfigMap for wrapper script (configure and runs the container)
oc create configmap start-sh --from-file=start.sh
# Create imagestream and buildconfig
oc create -f azure-imagestream.yaml -f azure-buildconfig.yaml
# Start new build
oc start-build azure-build-agent
# Create service account for build agent
oc create -f azure-sa.yaml
# Upload new SecurityContextConstraints for azure build agent
oc apply -f azure-sa-scc.yaml
# Apply scc to the service account
oc adm policy add-scc-to-user azure-build-scc -z azure-build-sa

# Configure Azure DevOps credentials
oc create secret generic azdevops \
  --from-literal=AZP_URL=https://dev.azure.com/yourOrg \
  --from-literal=AZP_TOKEN=YourPAT \
  --from-literal=AZP_POOL=NameOfYourPool

# Deploy the azure build agent
oc create -f azure-deployment.yaml
