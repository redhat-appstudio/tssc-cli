{{- $acs := required "Red Hat ACS settings" .Installer.Products.Advanced_Cluster_Security -}}
{{- $ingressDomain := required "OpenShift ingress domain" .OpenShift.Ingress.Domain -}}
{{- $ingressRouterCA := required "OpenShift RouterCA" .OpenShift.Ingress.RouterCA -}}
{{- $acsActive := true -}}
{{- if and (kindIs "bool" $acs.Enabled) (eq $acs.Enabled false) }}{{- $acsActive = false -}}{{- end -}}
---
acs: &acs
  enabled: {{ $acsActive }}
  name: &acsName stackrox-central-services
  ingressDomain: {{ $ingressDomain }}
  ingressRouterCA: {{ $ingressRouterCA }}
  integrationSecret:
    namespace: {{ .Installer.Namespace }}
  test:
    scanner:
      image: registry.access.redhat.com/ubi10:latest
  tssc:
    namespace: {{ .Installer.Namespace }}
acsTest: *acs
