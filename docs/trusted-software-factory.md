# Trusted Software Factory Installer

## Quickstart

1. Clone this repository/branch.
2. Copy `hack/private.env.template` to `konflux.env`.
3. Edit `konflux.env` and set the variables.
4. Get a clean OCP cluster (4.18 to 4.20).
5. From the terminal, log on that cluster.
6. Run `./hack/deploy.sh -e konflux.env -i github` and wait for the cluster to be configured.
7. Optionally, run `./hack/get-credentials.sh` to get the login information for the various services.
