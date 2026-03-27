# Script for tssc-cli repo

Copy **`run-periodic-integration.sh`** into the tssc-cli repository at:

**`integration-tests/scripts/periodic/run-periodic-integration.sh`**

See: https://github.com/redhat-appstudio/tssc-cli/tree/main/integration-tests/scripts/periodic

Konflux CronJobs in konflux-release-data fetch and run this script at runtime via:

```text
curl -sL https://raw.githubusercontent.com/redhat-appstudio/tssc-cli/main/integration-tests/scripts/periodic/run-periodic-integration.sh | bash
```
