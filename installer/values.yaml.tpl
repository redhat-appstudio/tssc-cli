{{- $crc := required "CRC settings" .Installer.Settings.crc -}}
{{- $tas := required "TAS settings" .Installer.Products.Trusted_Artifact_Signer -}}
{{- $acs := required "Red Hat ACS settings" .Installer.Products.Advanced_Cluster_Security -}}
{{- $gitops := required "GitOps settings" .Installer.Products.OpenShift_GitOps -}}
{{- $pipelines := required "Pipelines settings" .Installer.Products.OpenShift_Pipelines -}}
{{- $pipelinesNamespace := "openshift-pipelines" -}}
{{- $rhdh := required "RHDH settings" .Installer.Products.Developer_Hub -}}
{{- $ingressDomain := required "OpenShift ingress domain" .OpenShift.Ingress.Domain -}}
{{- $ingressRouterCA := required "OpenShift RouterCA" .OpenShift.Ingress.RouterCA -}}
{{- $openshiftMinorVersion := required "OpenShift Version" .OpenShift.MinorVersion -}}
{{- $authProvider := required "Auth Provider is required" $rhdh.Properties.authProvider }}
{{- $instNs := .Installer.Namespace -}}
{{- $tasActive := true -}}
{{- if and (kindIs "bool" $tas.Enabled) (eq $tas.Enabled false) }}{{- $tasActive = false -}}{{- end -}}
{{- $tasNs := default $instNs $tas.Namespace -}}
{{- $tpa := index .Installer.Products "Trusted_Profile_Analyzer" -}}
{{- $tpaIntegration := index .Installer.Integrations "tpa" -}}
{{- $tpaBundleMode := "product" -}}
{{- if not $tpa }}{{- $tpaBundleMode = "integration" -}}{{- end -}}
{{- if $tpa -}}
{{- if and (kindIs "bool" $tpa.Enabled) (eq $tpa.Enabled false) }}{{- $tpaBundleMode = "integration" -}}{{- end -}}
{{- end -}}
{{- $tpaActive := false -}}
{{- if eq $tpaBundleMode "product" -}}
{{- $tpaActive = true -}}
{{- if and $tpa (kindIs "bool" $tpa.Enabled) (eq $tpa.Enabled false) }}{{- $tpaActive = false -}}{{- end -}}
{{- end -}}
{{- $tpaNs := "tssc-tpa" -}}
{{- if eq $tpaBundleMode "product" -}}
{{- $tpaNs = default $instNs $tpa.Namespace -}}
{{- else if $tpa -}}
{{- $tpaNs = default "tssc-tpa" $tpa.Namespace -}}
{{- end -}}
{{- $tpaManageSub := false -}}
{{- if $tpa -}}{{- $tpaManageSub = dig "Properties" "manageSubscription" false $tpa -}}{{- end -}}
{{- $acsActive := true -}}
{{- if and (kindIs "bool" $acs.Enabled) (eq $acs.Enabled false) }}{{- $acsActive = false -}}{{- end -}}
{{- $acsNs := default $instNs $acs.Namespace -}}
{{- $gitopsActive := true -}}
{{- if and (kindIs "bool" $gitops.Enabled) (eq $gitops.Enabled false) }}{{- $gitopsActive = false -}}{{- end -}}
{{- $gitopsNs := default $instNs $gitops.Namespace -}}
{{- $pipelinesActive := true -}}
{{- if and (kindIs "bool" $pipelines.Enabled) (eq $pipelines.Enabled false) }}{{- $pipelinesActive = false -}}{{- end -}}
{{- $rhdhActive := true -}}
{{- if and (kindIs "bool" $rhdh.Enabled) (eq $rhdh.Enabled false) }}{{- $rhdhActive = false -}}{{- end -}}
{{- $rhdhNs := default $instNs $rhdh.Namespace -}}
{{- $keycloakEnabled := or $tpaActive $tasActive (and $rhdhActive (eq $authProvider "oidc")) (not (empty $tpaIntegration)) }}
{{- $keycloakNamespace := "tssc-keycloak" -}}
---
debug:
  ci: {{ dig "ci" "debug" false .Installer.Settings }}

#
# tssc-openshift
#

openshift:
  projects:
{{- if $keycloakEnabled }}
    - {{ $keycloakNamespace }}
    - rhbk-operator
{{- end }}
{{- if $acsActive }}
    - {{ $acsNs }}
    {{- if $acs.Properties.manageSubscription }}
    - rhacs-operator
    {{- end }}
{{- end }}
{{- if or $tpaActive $tpaIntegration }}
    - {{ $tpaNs }}
    {{- if and $tpaActive $tpaManageSub }}
    - rhtpa-operator
    {{- end }}
{{- end }}
{{- if $gitopsActive }}
    - {{ $gitopsNs }}
{{- end }}
{{- if $tasActive }}
    - {{ $tasNs }}
{{- end }}
{{- if $rhdhActive }}
    - {{ $rhdhNs }}
{{- end }}

#
# tssc-subscriptions
#


subscriptions:
  openshiftGitOps:
    enabled: {{ $gitopsActive }}
    managed: {{ and $gitopsActive $gitops.Properties.manageSubscription }}
    config:
      argoCDClusterNamespace: {{ $gitopsNs }}
  openshiftKeycloak:
    enabled: {{ $keycloakEnabled }}
    managed: {{ $keycloakEnabled }}
    operatorGroup:
      targetNamespaces:
        - {{ default "empty" $keycloakNamespace }}
  openshiftPipelines:
    enabled: {{ $pipelinesActive }}
    managed: {{ and $pipelinesActive $pipelines.Properties.manageSubscription }}
  openshiftTrustedArtifactSigner:
    enabled: {{ $tasActive }}
    managed: {{ and $tasActive $tas.Properties.manageSubscription }}
  trustedProfileAnalyzer:
    enabled: {{ $tpaActive }}
    managed: {{ and $tpaActive $tpaManageSub }}
  advancedClusterSecurity:
    enabled: {{ $acsActive }}
    managed: {{ and $acsActive $acs.Properties.manageSubscription }}
  developerHub:
    enabled: {{ $rhdhActive }}
    managed: {{ and $rhdhActive $rhdh.Properties.manageSubscription }}

#
# tssc-infrastructure
#

infrastructure:
  developerHub:
    namespace: {{ $rhdhNs }}
  pgsqlService:
    instances:
      - name: tpa
        enabled: {{ $tpaActive }}
        namespace: {{ $tpaNs }}
      - name: keycloak
        enabled: {{ $keycloakEnabled }}
        namespace: {{ $keycloakNamespace }}
  openShiftPipelines:
    enabled: {{ $pipelinesActive }}
    namespace: {{ $pipelinesNamespace }}

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
# tssc-integrations (also needs top-level .Values.argoCD for gitops auth templates)
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
