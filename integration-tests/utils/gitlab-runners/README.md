# Setting up self-managed runners on OpenShift

The default runner has CI/CD minute limits on the Free tier of GitLab.com, for more detail, you can refer to this link https://about.gitlab.com/blog/2020/09/01/ci-minutes-update-free-users/. To avoid that limit, we use self-managed runners on Red Hat OpenShift v4.


## Install `Gitlab Runner` operator

1. Create namespace: 

```
$ oc new-project gitlab-runner
```

2. Install `GitLab Runner` operator in cluster in namespace `gitlab-runner`

## Preparation for Gitlab Runner 

1. Create service account: 
```
oc apply -f gitlab-ci-sa.yml 
```

2. Create SCC for service account: 

```
$ oc apply -f gitlab-ci-sa-scc.yml 
```

3. Create custom config for runner: 

```
$oc create configmap custom-config-toml --from-file config.toml=custom-config-gitlab-ci.toml -n gitlab-runner
```

The above steps are one time, once they are ready, you can following up the following steps to create your own Gitlab Runner

# Configure your own Runner

1. Create Runner in GitLab.com:

* Login to https://gitlab.com, on the left sidebar, select `Groups`.
* Select a Group that you want to configure Gitlab Runner
* Select `Build` -> `Runner` on the left sidebar
* Click on `Create Group runner`, Tick on `Run untagged jobs`, 
* Click on `Create runner` and copy token from code snippet - something like "glrt-.....".

2. Make a copy of gitlab-runner-secret.yml with *difference* secret name, replace runner token that you get in step `1. Create Runner in GitLab.com:` and apply

```
$ oc apply -f gitlab-runner-secret.yml   
```

3. Make a copy of gitlab-runner.yml with *different* Runner name, edit `token` to specify the secret name you create at step 2 and apply 

```
$oc apply -f gitlab-runner.yml
```

4. Go to https://gitlab.com, Select the group you that you configured Gitlab Runner
5. on the left sidebar, select `Settings` --> `CI/CD`
6. Expand `Runners` and only enable the 3rd option `Allow members of projects and groups to create runners with runner registration tokens`