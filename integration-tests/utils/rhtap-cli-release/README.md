# RHTAP-CLI release

Consists of the script tssc-konflux.sh that automates the
creation of an MR to add new applicatoin version in konflux and
release new version application through konflux.

NOTE: rhtap-cli-stream.yaml is expected to be in order by version. Oldest version first entry
      in file and new version last entry in file.

## tssc-konflux.sh

Script updates the appropriate files and creates the MR to add
new application version to konflux and release new version. It is left to
the user to get the MR approved, merged and verify actions complete
successfully in konflux.

### Required Positional Arguments
VERSION required to be set see syntax

### Syntax
```console
user@machine:~$ ./tssc-konflux.sh -h

Usage:
    tssc-konflux.sh [options] <version>
       <version> = Application version to create and release on konflux.

Optional arguments:
    --dry-run
        Do not push updates and create MR.
    -d, --debug
        Activate tracing/debug mode.
    -h, --help
        Display this message.
    -k, --keep
        Number of Versions to keep. Will keep this many versions and
          delete the others. NOTE: <version> should be omitted if not creating
          new application and release version.
    -w, --wip
        Set work in progress, MR will be set as Draft
Example:
    tssc-konflux.sh 1.7
```

### Execution
1. Run script [./tssc-konflux.sh](./tssc-konflux.sh)

