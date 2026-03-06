# RHTAP Service Cluster Jenkins Cleanup

Cleaning up Jenkins job directories created from testing which have no builds run in the past X number
of days (Default: 14). The script has option to dry run for getting list of items that will be deleted without deleting them.
Options exits to force deleting of empty folders, items with no builds and to set number of days. See syntax below.

### Syntax
```console
user@machine:~$ ./service-jenkins-cleanup.sh -h

./service-jenkins-cleanup.sh - Cleans up old job directories that have no builds that have run
     in X number of days

./service-jenkins-cleanup.sh [options]

Note: Environment variables JENKINS_API_TOKEN is required to be set.
      Must set environment variables JENKINS_USERNAME and JENKINS_URL
           if not using defaults of 'cluster-admin-admin-edit-view' and
           'https://jenkins-jenkins.apps.rosa.rhtap-services.xmdt.p3.openshiftapps.com'

options:
-h, --help                  Show brief help
-d, --dry_run=dry_run       No actions actually performed, List of directories to remove. Valid values ['true', 'false'] Default: true
-o, --older=DAYS            Specify number of days old directories last modified, Default: 14
-e, --empty_folders         Delete empty folders. Default: false
-n, --no_builds             Delete if contains no builds. Default: false
-v, --verbose               Will output verbose info about processing. Default: false

```
### Manual Cleanup

1. Set environment variable `JENKINS_API_TOKEN`
   Must also set environment variables `JENKINS_USERNAME` and `JENKINS_URL`
       if not using defaults of 'cluster-admin-admin-edit-view' and 'https://jenkins-jenkins.apps.rosa.rhtap-services.xmdt.p3.openshiftapps.com'
2. Run script [./service-jenkins-cleanup.sh](./service-jenkins-cleanup.sh)

### Setup CronJob for cleanup in Konflux

1. Add cronjob.yml to tenants-config/cluster/stone-prd-rh01/tenants/rhtap-shared-team-tenant/periodics. Named appropriately
2. Add it to kustomization.yaml in tenants-config/cluster/stone-prd-rh01/tenants/rhtap-shared-team-tenant/periodics
3. Update README.md with cronjob info in tenants-config/cluster/stone-prd-rh01/tenants/rhtap-shared-team-tenant/periodics

### Set CronJob for regular cleanup in cluster

1. Login to OpenShift cluster where cronjob needs to be setup.
2. (Optional) Update Namespace value in [resources.yaml](./resources.yaml) if required. Default is set to `rhtap-cleanup`
3. Provide the service jenkins credentials `JENKINS_USERNAME`, `JENKINS_API_TOKEN` and `JENKINS_URL` for Secret resource in [resources.yaml](./resources.yaml)
4. (Optional) Update schedule value for CronJob resource in [resources.yaml](./resources.yaml). Default is set to run every Saturday 7:00 AM
5. Create resources for setting CronJob by running: `oc apply -f resources.yaml`
