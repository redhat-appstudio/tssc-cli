{{- $pipelines := required "Pipelines settings" .Installer.Products.OpenShift_Pipelines -}}
{{- $pipelinesNamespace := "openshift-pipelines" -}}
{{- $pipelinesActive := true -}}
{{- if and (kindIs "bool" $pipelines.Enabled) (eq $pipelines.Enabled false) }}{{- $pipelinesActive = false -}}{{- end -}}
---
pipelines_config:
  namespace:  {{ $pipelinesNamespace }}
  integrationSecret:
    namespace: {{ .Installer.Namespace }}

# Required by templates/tektonconfig/tektonconfig.yaml (.Values.infrastructure.openShiftPipelines.enabled)
infrastructure:
  openShiftPipelines:
    enabled: {{ $pipelinesActive }}
    namespace: {{ $pipelinesNamespace }}
