Contributing to `rhtap-cli`
---------------------------

In order to contribute to this project you need the following requirements:

- [Golang 1.16 or higher][golang]
- [GNU Make][gnuMake]
- [GNU Tar][gnuTar]
- [Podman][podman] or [Buildah][buildah] (optional)

All the automation needed for the project lives in the [Makefile](Makefile). This file is the entry point for all the automation tasks in the project for CI, development and release

For macOS users make sure `gtar` is available, i.e.:

```bash
brew install gnu-tar
```

# Building

After you have cloned the repository and installed the requirements, you can start building. For that purpose you can simply run `make`, the default target builds the application in the `bin` directory

```bash
make
```

The resources needed for the installer are stored in the `installer` directory, you can find the `config.yaml` file there. Before the application is built, the contents of the installer directory are packaged into a tarball and embedded into the application binary.

To package the installer resources into a tarball run the following target:

```bash
make installer-tarball
ls -l installer/installer.tar
```

The `installer/installer.tar` is intentionally excluded from version control.

## Container Image

In order to build a container image out of this project run the following target:

```bash
make image IMAGE_REPO="ghcr.io/redhat-appstudio/rhtap-cli" IMAGE_TAG="latest"
```

The `IMAGE_REPO` and `IMAGE_TAG` are optional variables, you should use your own repository and tag for the image.

By default the container is build using `podman`, you can alternatively use `buildah` by running the target `make image-buildah` instead.

# Testing

Unit testing is done using the `go test` command, you can run the tests with the following target:

```bash
make test-unit
```

Alternatively, run all tests with:

```bash
make test
```

# Running

To run the application you can rely on the `run` target, this is the equivalent of `go run` command. For instance:

```bash
make run ARGS='deploy --help'
```

Which is the equivalent of building and running the application, i.e.:

```bash
make &&
    bin/rhtap-cli deploy --help
```

# GitHub Release

This project uses [GitHub Actions](.github/workflows/release.yaml) to automate the release process, triggered by a new tag in the repository.

To release this application using the the GitHub web interface follow the steps:

1. Go to the [releases page][releases]
2. Click on "Create a new release" button
3. Choose the tag you want to release, the tag must start with `v` and follow the semantic versioning pattern.
4. Fill the release title and description
5. [Wait for the release workflow][actions] to finish and verify the release assets

## Release Automation

### Workflow

For the release automation the following tools are used:
- [`gh`][gitHubCLI]: GitHub helper CLI, ensure the release is created, or create it if it doesn't exist yet.
- [`goreleaser`][goreleaser]: Tool to automate the release process, it creates the release assets and uploads them to the GitHub release.

The [release workflow](.github/workflows/release.yaml) relies on the `make github-release` target, this [`Makefile`](Makefile) target is responsible for ensure the release is created, or create it using `gh` helper, build and upload the release assets using `goreleaser`.

The GitHub workflow provides [`GITHUB_REF_NAME` environment variable][gitHubDocWorkflowEnvVars] to the release job, this variable is used to determine the tag name to release.

```bash
make github-release GITHUB_REF_NAME="v0.1.0"
```

### Assets

The release assets are built using `goreleaser`, the configuration for this tool is stored in the [`.goreleaser.yaml`](.goreleaser.yaml) file.

To build the release assets for only the current platform, run:

```bash
make snapshot ARGS='--single-target --output=bin/rhtap-cli'
```

To build the release assets for all platforms, run:

```bash
make snapshot
```

[actions]: https://github.com/redhat-appstudio/rhtap-cli/actions
[gitHubCLI]: https://cli.github.com
[gitHubDocWorkflowEnvVars]: https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/variables#default-environment-variables
[gnuMake]: https://www.gnu.org/software/make
[golang]: https://golang.org/dl
[goreleaser]: https://goreleaser.com
[buildah]: https://buildah.io
[podman]: https://podman.io
[releases]: https://github.com/redhat-appstudio/rhtap-cli/releases
[gnuTar]: https://www.gnu.org/software/tar
