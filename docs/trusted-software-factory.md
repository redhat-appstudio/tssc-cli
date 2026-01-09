# Trusted Software Factory Installer

## Quickstart

1. Clone this repository/branch.
2. Copy `hack/private.env.template` to `konflux.env`.
3. Edit `konflux.env` and set the variables.
4. Get a clean OCP cluster (4.18 to 4.20).
5. From the terminal, log on that cluster.
6. Run `./hack/deploy.sh -e konflux.env -i github` and wait for the cluster to be configured.
7. Optionally, run `./hack/get-credentials.sh` to get the login information for the various services.

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
