# Trusted Software Factory Installer

## Quickstart

1. Copy [private.env.template](../hack/private.env.template) as `tsf.env` and fill in the blanks.
2. Start a container with `podman run -it --rm --env-file tsf.env --entrypoint bash -p 8228:8228 --pull always quay.io/roming22-org/tsf:latest`.
3. Log on the cluster with `oc login "$OCP__API_ENDPOINT" --username "$OCP__USERNAME" --password "$OCP__PASSWORD"`
4. Create the TSF config on the cluster with `tssc config --create`.
5. Check if the `Red Hat Cert-Manager` operator is already installed in the cluster. If it is, edit the `tssc-config` ConfigMap in the `tssc` namespace to set `manageSubscription: false` for that product.
5. Create the github app integration with `tssc integration github --create --org "$GITHUB__ORG" "tsf-$(date +%m%d-%H%M)"`. Open the link to create the app, follow the instructions, and install the application to your GitHub organization.
6. Create the quay integration with `tssc integration quay --organization="$QUAY__ORG" --token="$QUAY__API_TOKEN" --url="$QUAY__URL"`. If you need information on how to generate the token, look further in this document.
7. Deploy all the services with `tssc deploy`.
8. Cluster users (including the admin user) should now be able to login the Konflux UI. You will find the URL in the logs of the deployment. You can also access it through the GitHub App (via the `Website` link on the application public page, or via the link displayed on the application configuration page which was the last page displayed after you installed the application).

## Getting the information for tsf.env

### GitHub

If you don't have one, create a test organization.
Use the name of that organization for `GITHUB__ORG`.

### OpenShift

* `OCP__API_ENDPOINT`: the full url of the cluster's api endpoint. Example: ``.
* `OCP__USERNAME`: User with admin privileges on the cluster. Example: `admin`.
* `OCP__PASSWORD`: Credential for the user.

### Quay

* `QUAY__URL`: full url of the quay instance. Example: `https://quay.io`.
* `QUAY__API_TOKEN`: token giving access to an organization on the quay instance.
* `QUAY__ORG`: the organization that the token gives access to.

To create the API token:
* Go to quay homepage.
* On the right hand side, click on the organization you wish to use. The use of an organization is mandatory. If you do not have one, you can create one by clicking `Create New Organization`.
* On the left hand side, click on the `Applications` icon (second to last icon).
* On the right hand side, click on `Create New Application`, and enter the application name (e.g. `tsf`).
* Click the name of the newly created application.
* On the left hand side, click on the `Generate Token` icon (last icon).
* Select everything (TODO: find the minimum set of permissions required).
* Click on `Generate Access Token`.
* Click on `Authorize Application`.
* Copy the access token.

## Services

### Quay

When a new component is created in Konflux, a new repository is created, in the organization specified at install time.
If you are using a free quay.io account, the visibility of the repository should be changed to public manually because of the account limitations.
If you are use a corporate account, or your own Quay instance, the repository visibility can remain private.

## Troubleshooting

### tssc-subscription fails to install with "Error: upgrade failed"

#### Full error

```
Error: upgrade failed: Unable to continue with update: Subscription "openshift-cert-manager-operator" in namespace "cert-manager-operator" exists and cannot be imported into the current release: invalid ownership metadata; label validation error: missing key "app.kubernetes.io/managed-by": must be set to "Helm"; annotation validation error: missing key "meta.helm.sh/release-name": must be set to "tssc-subscriptions"; annotation validation error: missing key "meta.helm.sh/release-namespace": must be set to "tssc"
```

#### Root cause

The subscription has already been installed by a third party. Helm does not want to take ownership of the resource.

#### Workaround

* Add `-i cert-manager` to the `./hack/tssc deploy` command.
