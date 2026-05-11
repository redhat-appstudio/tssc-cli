{{- $_ := required "Red Hat ACS settings" .Installer.Products.Advanced_Cluster_Security -}}
---
# Same acsTest shape as charts/tssc-acs/values.yaml.tpl (this chart uses its own ReadValuesTemplate).
acsTest:
  name: stackrox-central-services
  integrationSecret:
    namespace: {{ .Installer.Namespace }}
  test:
    scanner:
      image: registry.access.redhat.com/ubi10:latest
  tssc:
    namespace: {{ .Installer.Namespace }}
