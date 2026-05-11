{{- /*
  Select the enabled subscriptions.
*/ -}}
{{- define "subscriptions.enabled" -}}
  {{- $enabled := dict -}}
  {{- range $k, $v := .Values.subscriptions -}}
    {{- if $v.enabled -}}
      {{- $enabled = merge $enabled (dict $k $v) -}}
    {{- end -}}
  {{- end -}}
  {{- $enabled | toYaml -}}
{{- end -}}
{{- /*
  Select the managed subscriptions.
*/ -}}
{{- define "subscriptions.managed" -}}
  {{- $managed := dict -}}
  {{- range $k, $v := .Values.subscriptions -}}
    {{- if $v.managed -}}
      {{- $managed = merge $managed (dict $k $v) -}}
    {{- end -}}
  {{- end -}}
  {{- $managed | toYaml -}}
{{- end -}}
