#!/bin/bash
set -e

# Create new project 'jenkins'
oc new-project jenkins

# Deploy jenkins from openshift template
oc new-app -e JENKINS_PASSWORD=admin123 -e VOLUME_CAPACITY=10Gi jenkins-persistent

sleep 15  # it could take some time, until jenkins pod is created

# Upload new SecurityContextConstraints for Jenkins agent (for running jenkins agent in privileged mode)
oc apply -f ./jenkins-agent/jenkins-agent-base-scc.yml
# Apply policy to Jenkins user (for running jenkins agent in privileged mode)
oc adm policy add-scc-to-user jenkins-agent-base -z jenkins
oc adm policy add-scc-to-user privileged -z jenkins -n jenkins
