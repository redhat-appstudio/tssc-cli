{{- $gitops := required "GitOps settings" .Installer.Products.OpenShift_GitOps -}}
{{- $ingressDomain := required "OpenShift ingress domain" .OpenShift.Ingress.Domain -}}
{{- $argoCDName := printf "%s-gitops" .Installer.Namespace }}
{{- $instNs := .Installer.Namespace -}}
{{- $gitopsActive := true -}}
{{- if and (kindIs "bool" $gitops.Enabled) (eq $gitops.Enabled false) }}{{- $gitopsActive = false -}}{{- end -}}
{{- $gitopsNs := default $instNs $gitops.Namespace -}}
# Bundle-scoped openshift/subscriptions for tssc-gitops-openshift / tssc-gitops-subscriptions.
openshift:
  projects:
    - {{ $gitopsNs }}
subscriptions:
  openshiftGitOps:
    enabled: {{ $gitopsActive }}
    managed: {{ and $gitopsActive $gitops.Properties.manageSubscription }}
    config:
      argoCDClusterNamespace: {{ $gitopsNs }}
argoCD:
  enabled: {{ $gitopsActive }}
  name: {{ $argoCDName }}
  namespace: {{ $gitopsNs }}
  integrationSecret:
    name: tssc-argocd-integration
    namespace: {{ .Installer.Namespace }}
  ingressDomain: {{ $ingressDomain }}
  tssc:
    namespace: {{ .Installer.Namespace }}
