{{/*

  Returns the fully qualified domain name for the ArgoCD server.

*/}}
{{- define "argoCD.serverHostname" -}}
  {{- $argoCD := .Values.argoCD -}}
  {{ printf "%s-server-%s.%s" $argoCD.name $argoCD.namespace $argoCD.ingressDomain }}
{{- end -}}

{{/*

  Returns the name of the secret that contains the ArgoCD admin password.

*/}}
{{- define "argoCD.secretClusterName" -}}
  {{ printf "%s-cluster" .Values.argoCD.name }}
{{- end -}}
