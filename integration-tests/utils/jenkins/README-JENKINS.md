# RHTAP QE Jenkins Guide

## Setup Jenkins on OpenShift

1. Login to OpenShift
2. `cd` to the directory that contains this README
3. (Optional) You can comment lines related to building image in file [./jenkins-deploy.sh](./jenkins-deploy.sh) and use prebuild image on Quay in Step 8 Option B
4. Run the [./jenkins-deploy.sh](./jenkins-deploy.sh) script
5. (Optional) You can check if the pipeline uses the correct agent image and whether buildah is working
    - In your Jenkins instance, click `+ New Item` and create a `Pipeline` style project
    - Use the content of the [pipeline-buildah](./pipeline-buildah) file as the pipeline script
    - Save the pipeline and trigger a new build. It should succeed and print the buildah help message
6. Login in the Jenkins instance and get your username and token
    - You can see the username when you click on your user in the top right corner (like `jkopriva@redhat.com-admin-edit-view`, `cluster-admin-admin-edit-view`)
    - In the user menu, click on `Configure` and generate an API token
    - You will need the username and token for RHTAP installation, setting creds in Jenkins, running e2e tests...
7. You can continue with RHTAP Installation (do not forget to add Jenkins integration with correct parameters: Jenkins URL, username and token)
8. When you are creating a Job in Jenkins, it needs to have same the same name as the corresponding component to make RHDH associate them. For source repo job use the <component-name> for gitops repo job use <component-name>-gitops

## Setup credentials for RHTAP Jenkins on OpenShift for testing

1. To populate all credential required by Jenkins job run [jenkins-credentials.sh](./jenkins-credentials.sh) script
    - Make sure you have jenkins integration secret in place
    - Make sure you have deployed tssc using the installer

## Common issues

 - I do not see Jenkins during creation of component - You have wrong URL to catalog URL, you can change it here: `ns/tssc-dh/secrets/tssc-developer-hub-env/yaml` and kill old developer hub pods and wait for new ones
 - Buildah in jenkins job/pipeline does not work - You have not changed agent in Jenkinsfile or the Jenkins agent pod does not run in privileged mode
 - Jenkins job status is not reported back to developer hub - You do not have correct credentials for Jenkins in developer hub, update `ns/tssc-dh/secrets/tssc-developer-hub-env/yaml` with correct creds
 - `error: error processing template "openshift/jenkins-persistent": the namespace of the provided object does not match the namespace sent on the request`
   when running `oc new-app` - your `oc` version may be out of date, try downloading the latest version
