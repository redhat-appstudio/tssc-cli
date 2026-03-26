#!/usr/bin/env bash
# Run periodic integration job: trigger scheduler, wait for PipelineRun, send Slack.
# Hosted in tssc-cli (integration-tests/scripts/periodic/). Invoked by Konflux CronJobs in konflux-release-data
# via: curl -sL https://raw.githubusercontent.com/redhat-appstudio/tssc-cli/main/integration-tests/scripts/periodic/run-periodic-integration.sh | bash
# Configure via environment variables; see defaults below.

set -euo pipefail

# Required
: "${PERIODIC_SCENARIO_NAME:?PERIODIC_SCENARIO_NAME must be set}"

# Optional with defaults
PERIODIC_KONFLUX_TENANT_NAME="${PERIODIC_KONFLUX_TENANT_NAME:-rhtap-shared-team-tenant}"
PERIODIC_KONFLUX_APP_NAME="${PERIODIC_KONFLUX_APP_NAME:-tssc-cli}"
PERIODIC_KONFLUX_COMPONENT_NAME="${PERIODIC_KONFLUX_COMPONENT_NAME:-tssc-cli}"
PERIODIC_REPO_URL="${PERIODIC_REPO_URL:-https://github.com/redhat-appstudio/tssc-cli.git}"
PERIODIC_BRANCH="${PERIODIC_BRANCH:-main}"
PERIODIC_SCHEDULE_DAYS="${PERIODIC_SCHEDULE_DAYS:-5d}"
PERIODIC_LOGS_APP_NAME="${PERIODIC_LOGS_APP_NAME:-rhtap-task-runner}"
PERIODIC_MSG_LABEL="${PERIODIC_MSG_LABEL:-tssc-ci-weekly job}"
KONFLUX_UI_BASE_URL="${KONFLUX_UI_BASE_URL:-https://konflux-ui.apps.stone-prd-rh01.pg1f.p1.openshiftapps.com}"

# Optional: extra args for the scheduler (e.g. --no-old-commits-run or install params for RHDH)
PERIODIC_SCHEDULER_EXTRA_ARGS="${PERIODIC_SCHEDULER_EXTRA_ARGS:-}"
read -r -a PERIODIC_SCHEDULER_EXTRA_ARGS_ARR <<< "${PERIODIC_SCHEDULER_EXTRA_ARGS}"

waitFor() {
  local PLR="$1"
  local NAMESPACE="$2"
  local MSG_RUNNING="$3"
  local MSG_DONE="$4"

  echo "[INFO] Waiting for PipelineRun '${PLR}' to complete..."
  timeout --foreground 120m bash -c "
    while true; do
      status=\$(oc get pipelinerun/${PLR} -n ${NAMESPACE} -o jsonpath='{.status.conditions[?(@.type==\"Succeeded\")].status}')
      if [[ \"\$status\" == \"True\" || \"\$status\" == \"False\" ]]; then
        echo \"$MSG_DONE\"
        break
      else
        echo \"$MSG_RUNNING\"
        sleep 60
      fi
    done
  "
}

curl -sL https://raw.githubusercontent.com/konflux-ci/tekton-integration-catalog/main/scripts/integration-tests-scheduler/integration-tests-scheduler.sh | bash -s -- \
  --repository-url "${PERIODIC_REPO_URL}" \
  --branch "${PERIODIC_BRANCH}" \
  --konflux-scenario-name "${PERIODIC_SCENARIO_NAME}" \
  --konflux-tenant-name "${PERIODIC_KONFLUX_TENANT_NAME}" \
  --konflux-application-name "${PERIODIC_KONFLUX_APP_NAME}" \
  --konflux-component-name "${PERIODIC_KONFLUX_COMPONENT_NAME}" \
  --schedule "${PERIODIC_SCHEDULE_DAYS}" \
  "${PERIODIC_SCHEDULER_EXTRA_ARGS_ARR[@]}"

sleep 15

LATEST_PIPELINE_RUN=$(tkn pipelinerun list -n "${PERIODIC_KONFLUX_TENANT_NAME}" -o json | jq -r --arg scenario "${PERIODIC_SCENARIO_NAME}" '
  .items
  | map(select(
      .metadata.labels."test.appstudio.openshift.io/run" == $scenario and
      (
        (.status.conditions // [])
        | map(select(
            .type == "Succeeded" and
            .status == "Unknown"
          ))
        | length > 0
      )
    ))
  | sort_by(.metadata.creationTimestamp)
  | last
  | .metadata.name // empty
')

if [[ -z "$LATEST_PIPELINE_RUN" ]]; then
  echo "[ERROR] No matching running PipelineRun found for scenario: $PERIODIC_SCENARIO_NAME"
  echo "[DEBUG] Consider listing all PipelineRuns with: tkn pipelinerun list -n ${PERIODIC_KONFLUX_TENANT_NAME}"
  exit 1
fi
echo "[INFO] Found running PipelineRun: $LATEST_PIPELINE_RUN"

waitFor "$LATEST_PIPELINE_RUN" "$PERIODIC_KONFLUX_TENANT_NAME" "Nested pipelines are still running. Waiting 1 minute" "All nested pipelines finished"

echo "[INFO] Getting pipeline run status and logs URL"
status=$(oc get "pipelinerun/${LATEST_PIPELINE_RUN}" -n "${PERIODIC_KONFLUX_TENANT_NAME}" -o jsonpath="{.status.conditions[].message}")
logsUrl="${KONFLUX_UI_BASE_URL}/ns/${PERIODIC_KONFLUX_TENANT_NAME}/applications/${PERIODIC_LOGS_APP_NAME}/pipelineruns/${LATEST_PIPELINE_RUN}"

echo "[INFO] Setting icon and run status in message"
MSG="<icon> *${PERIODIC_MSG_LABEL}* pipeline run <pr_name> *<run_status>* <icon> <<logs_url>|View logs>"
if [[ "${status}" == *"Failed: 0"* ]]; then
  MSG="${MSG//<icon>/:success-kid:}"
  MSG="${MSG//<run_status>/succeeded}"
else
  MSG="${MSG//<icon>/:fuuuuuu:}"
  MSG="${MSG//<run_status>/failed}"
fi
MSG="${MSG//<logs_url>/$logsUrl}"
MSG="${MSG//<pr_name>/$LATEST_PIPELINE_RUN}"

echo "[INFO] Sending Slack notification"
curl --no-progress-meter -X POST -H 'Content-Type: application/json' \
  --data "{\"blocks\":[{\"type\":\"section\",\"text\":{\"type\":\"mrkdwn\",\"text\":\"${MSG}\"}}]}" "$SLACK_WEBHOOK_URL"
