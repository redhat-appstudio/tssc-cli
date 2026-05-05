{{- $pipelinesNamespace := "openshift-pipelines" -}}
---
pipelines:
  namespace: {{ $pipelinesNamespace }}
  tssc:
    namespace: {{ .Installer.Namespace }}
