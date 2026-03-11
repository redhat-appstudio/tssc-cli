# E2E integration tests

This folder contains resources for running multi-level e2e for TSSC in Konflux

## Prerequisites
In order to succesfully run this integration test one must:
* Create [Role](./custom-resources/plrManager-Role.yaml) that is capable of starting & watching PipelineRuns 
* Create a [RoleBinding](./custom-resources/plrManager-RoleBinding.yaml) binding that role to `appstudio-pipeline` SA (default SA used by Tekton in Konflux)
* Create [IntegrationTestScenario](./custom-resources/e2e-IntegrationTestScenario.yaml) in Konflux tenant.
* Create necessary secrets in the Konflux tenant
    * `sprayproxy-auth` used to register ephemeray cluster with sprayproxy with keys:
        * `server-token`
        * `server-url`
    * `rhtap-cli-install` used by `rhtap-install` task with keys:

        * `docker_io: `
        * `rhdh-github-client-id`
        * `acs-central-endpoint`
        * `acs-api-token`
        * `rhdh-github-client-secret`
        * `bitbucket-username`
        * `bitbucket-token`
        * `rhdh-github-webhook-secret`
        * `github-username`
        * `github-password`
        * `github-2fa-secret`
        * `github_token`
        * `rhdh-github-app-id`
        * `rhdh-github-private-key`
        * `gitlab-app-id`
        * `gitlab-app-secret`
        * `gitlab-group`
        * `gitlab_token`
        * `jenkins-username`
        * `jenkins-url`
        * `jenkins-api-token`
        * `artifactory-dockerconfig-json`
        * `artifactory-token`
        * `artifactory-url`
        * `nexus-dockerconfig-json`
        * `nexus-ui-url`
        * `quay-dockerconfig-json`
        * `quay-api-token`

    * `konflux-test-infra` with keys:
        * `qe-cloud-credentials-us-east-1` - this is current default key holding AWS credentials json. Other key might be used based on what's defined in `cloud-credential-key` parameter of `rhtap-cli-e2e` Pipeline.
        * `github-bot-commenter-token`
        * `oci-storage`

## Pipelines used

### e2e-main-pipeline

This is the main pipeline used by our IntegrationTestScenario. It has 3 phases:
1) Gets test metadata and processes the rhads-config file from the repository to extract OpenShift versions and other configuration
2) Creates new PipelineRun for each OpenShift version specified in the rhads-config using [rhtap-cli-e2e Pipeline](./pipelines/rhtap-cli-e2e.yaml)
3) Waits for all "nested" pipelines to finish and reports their status

The pipeline automatically determines which repository and branch to use based on the job specification - for rhtap-cli repository changes, it uses the PR's repository and branch; for other repositories, it uses the default redhat-appstudio/rhtap-cli repository with the main branch.

### rhtap-cli-e2e

Pipeline that provisions infrastructure and runs end-to-end tests for RHADS. Given parameters, it:
1) Creates ephemeral ROSA cluster on AWS
2) Installs TSSC using the installer from the appropriate repository version
3) Runs comprehensive end-to-end tests using the `tssc-e2e` task, which validates the entire TSSC system functionality including:
   - Application onboarding workflows
   - CI/CD pipeline execution
   - Security scanning integration
   - Supply chain validation
   - Integration with external services (GitHub, Quay, etc.)
4) Teardown (archives artifacts, destroys cluster, cleans up resources)
