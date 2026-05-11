`tssc-pipelines-subscriptions`
---------------------

This Helm Chart manages OpenShift Operator Hub subscriptions required for TSSC. It leverages `Subscription` and `OperatorGroup` resources.

During installation, it checks for the existence of the necessary CRDs (Custom Resource Definitions) and waits for the Operator to become available before proceeding.

The list of subscriptions is defined in the [`values.yaml`](values.yaml) file.
