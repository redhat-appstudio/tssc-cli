{{- $pipelines := required "Pipelines settings" .Installer.Products.OpenShift_Pipelines -}}
{{- $pipelinesNamespace := "openshift-pipelines" -}}
{{- $pipelinesActive := true -}}
{{- if and (kindIs "bool" $pipelines.Enabled) (eq $pipelines.Enabled false) }}{{- $pipelinesActive = false -}}{{- end -}}
# Single YAML document (no "---"): chartutil.ReadValues unmarshals only the first document.
# Bundle-scoped openshift/subscriptions for tssc-pipelines-openshift / tssc-pipelines-subscriptions.
openshift:
  projects:
    - {{ $pipelinesNamespace }}
subscriptions:
  openshiftPipelines:
    enabled: {{ $pipelinesActive }}
    managed: {{ and $pipelinesActive $pipelines.Properties.manageSubscription }}
pipelines:
  namespace: {{ $pipelinesNamespace }}
  tssc:
    namespace: {{ .Installer.Namespace }}
pipelines_config:
  namespace:  {{ $pipelinesNamespace }}
  integrationSecret:
    namespace: {{ .Installer.Namespace }}

# Required by templates/tektonconfig/tektonconfig.yaml (.Values.infrastructure.openShiftPipelines.enabled)
infrastructure:
  openShiftPipelines:
    enabled: {{ $pipelinesActive }}
    namespace: {{ $pipelinesNamespace }}
