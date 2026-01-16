{{/*

  Returns the name of the Quay Registry instance's config secret.

*/}}
{{- define "quay.configSecretName" -}}
  {{- printf "%s-config" .Values.quay.name }}
{{- end -}}

{{/*

  Returns the name of the Quay Registry instance's super user secret.

*/}}
{{- define "quay.superUserSecretName" -}}
  {{- printf "%s-super-user" .Values.quay.name }}
{{- end -}}

{{/*

  Returns the FQDN for the Quay Registry instance deployed.

*/}}
{{- define "quay.registryHostname" -}}
  {{- printf "%s-quay-%s.%s"
        .Values.quay.name
        .Values.quay.namespace
        .Values.quay.ingressDomain
  -}}
{{- end -}}

{{/*

  Quay "quay_config.yaml" file template. Uses the configured S3 storage
  credentials secret to populate the S3 storage configuration.

*/}}
{{- define "quay.s3storage.configYAML" -}}
  {{- $quay := .Values.quay -}}
BROWSER_API_CALLS_XHR_ONLY: false
FEATURE_USER_CREATION: false
  {{- with $quay.config.superUser }}
FEATURE_USER_INITIALIZE: true
SUPER_USERS:
  - {{ .name }}
  {{- end }}
{{- end -}}
