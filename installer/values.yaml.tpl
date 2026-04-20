{{- $acs := required "Red Hat ACS settings" .Installer.Products.Advanced_Cluster_Security -}}
{{- $gitops := required "GitOps settings" .Installer.Products.OpenShift_GitOps -}}
{{- $pipelines := required "Pipelines settings" .Installer.Products.OpenShift_Pipelines -}}
{{- $instNs := .Installer.Namespace -}}
{{- $acsActive := true -}}
{{- if and (kindIs "bool" $acs.Enabled) (eq $acs.Enabled false) }}{{- $acsActive = false -}}{{- end -}}
{{- $gitopsActive := true -}}
{{- if and (kindIs "bool" $gitops.Enabled) (eq $gitops.Enabled false) }}{{- $gitopsActive = false -}}{{- end -}}
{{- $gitopsNs := default $instNs $gitops.Namespace -}}
{{- $pipelinesActive := true -}}
{{- if and (kindIs "bool" $pipelines.Enabled) (eq $pipelines.Enabled false) }}{{- $pipelinesActive = false -}}{{- end -}}
{{- $settings := .Installer.Settings | default dict -}}
{{- $ci := index $settings "ci" | default dict -}}
---
debug:
  ci: {{ default false (index $ci "debug") }}

#
# Global charts: tssc-app-namespaces. Bundle charts: tssc-gitops-integrations (gitops), tssc-acs-integrations (acs).
# Quay: use `tssc integration quay` for tssc-quay-integration Secret.
# Keycloak / IAM (openshift projects, subscription, PostgreSQL, realm) live under bundles/iam/.
#

#
# tssc-app-namespaces
#

{{- $argoCDName := printf "%s-gitops" .Installer.Namespace }}

appNamespaces:
  argoCD:
    name: {{ $argoCDName }}
  namespace_prefixes:
    - {{ printf "%s-app" .Installer.Namespace }}
  pipelines:
    enabled: {{ $pipelinesActive }}

#
# tssc-gitops-integrations / tssc-acs-integrations (top-level .Values.argoCD + .Values.integrations)
#

argoCD:
  enabled: {{ $gitopsActive }}
  namespace: {{ $gitopsNs }}

integrations:
  acs:
    enabled: {{ $acsActive }}
  argoCD:
    enabled: {{ $gitopsActive }}
    namespace: {{ $gitopsNs }}
  tssc:
    namespace: {{ .Installer.Namespace }}
#   github:
#     clientId: ""
#     clientSecret: ""
#     id: ""
#     host: "github.com"
#     publicKey: |
#       -----BEGIN RSA PRIVATE KEY-----   # notsecret
#       -----END RSA PRIVATE KEY-----     # notsecret
#     token: ""
#     webhookSecret: ""
#   gitlab:
#     token: ""
