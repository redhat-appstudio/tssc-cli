# RHTAP GitHub Organization Cleanup

Cleaning up github repositories created from rhtap-e2e tests which has no activity in past 2 weeks.
The script has option to dry run for getting the list of repos that will be deleted without deleting them.

### Manual Cleanup

1. Set environment variables `GITHUB_ORG_TOKEN` and `GITHUB_ORG_NAME`
2. Run script [./git-repo-cleanup.sh](./git-repo-cleanup.sh)


### Set CronJob for regular cleanup

1. Login to OpenShift cluster where cronjob needs to be setup.
2. (Optional) Update Namespace value in [resources.yaml](./resources.yaml) if required. Default is set to `rhtap-cleanup`
3. Provide your github credentials `GITHUB_ORG_TOKEN` and `GITHUB_ORG_NAME` for Secret resource in [resources.yaml](./resources.yaml)
4. (Optional) Update schedule value for CronJob resource in [resources.yaml](./resources.yaml). Default is set to run every Saturday 8:00 AM
4. Create resources for setting CronJob by running: `oc apply -f resources.yaml`
