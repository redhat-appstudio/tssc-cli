{{/*

  Returns the ACS URL for the given environment.

*/}}
{{- define "acs.centralEndPoint" -}}
  {{- $acs := .Values.acs -}}
  {{- printf "central-%s.%s" .Release.Namespace $acs.ingressDomain -}}
{{- end -}}
