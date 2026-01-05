{{- $crc := required "CRC settings" .Installer.Settings.crc -}}
{{- $tas := required "TAS settings" .Installer.Products.Trusted_Artifact_Signer -}}
{{- $tpa := required "TPA settings" .Installer.Products.Trusted_Profile_Analyzer -}}
{{- $gitops := required "GitOps settings" .Installer.Products.OpenShift_GitOps -}}
{{- $pipelines := required "Pipelines settings" .Installer.Products.OpenShift_Pipelines -}}
{{- $pipelinesNamespace := "openshift-pipelines" -}}
{{- $ingressDomain := required "OpenShift ingress domain" .OpenShift.Ingress.Domain -}}
{{- $ingressRouterCA := required "OpenShift RouterCA" .OpenShift.Ingress.RouterCA -}}
{{- $openshiftMinorVersion := required "OpenShift Version" .OpenShift.MinorVersion -}}
{{- $keycloakEnabled := or $tpa.Enabled $tas.Enabled -}}
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
{{- if $tpa.Enabled }}
    - {{ $tpa.Namespace }}
    {{- if $tpa.Properties.manageSubscription }}
    - rhtpa-operator
    {{- end }}
{{- end }}
{{- if $gitops.Enabled }}
    - {{ $gitops.Namespace }}
{{- end }}
{{- if $tas.Enabled }}
    - {{ $tas.Namespace }}
{{- end }}
{{- if $tpa.Enabled }}
    - {{ $tpa.Namespace }}
{{- end }}

#
# tssc-subscriptions
#


subscriptions:
  openshiftGitOps:
    enabled: {{ $gitops.Enabled }}
    managed: {{ and $gitops.Enabled $gitops.Properties.manageSubscription }}
    config:
      argoCDClusterNamespace: {{ $gitops.Namespace }}
  openshiftKeycloak:
    enabled: {{ $keycloakEnabled }}
    managed: {{ $keycloakEnabled }}
    operatorGroup:
      targetNamespaces:
        - {{ default "empty" $keycloakNamespace }}
  openshiftPipelines:
    enabled: {{ $pipelines.Enabled }}
    managed: {{ and $pipelines.Enabled $pipelines.Properties.manageSubscription }}
  openshiftTrustedArtifactSigner:
    enabled: {{ $tas.Enabled }}
    managed: {{ and $tas.Enabled $tas.Properties.manageSubscription }}
  trustedProfileAnalyzer:
    enabled: {{ $tpa.Enabled }}
    managed: {{ and $tpa.Enabled $tpa.Properties.manageSubscription }}

#
# tssc-infrastructure
#

infrastructure:
  pgsqlService:
    instances:
      - name: tpa
        enabled: {{ $tpa.Enabled }}
        namespace: {{ $tpa.Namespace }}
      - name: keycloak
        enabled: {{ $keycloakEnabled }}
        namespace: {{ $keycloakNamespace }}
  openShiftPipelines:
    enabled: {{ $pipelines.Enabled }}
    namespace: {{ $pipelinesNamespace }}

#
# tssc-iam
#

{{- $keycloakRouteTLSSecretName := "keycloak-tls" }}
{{- $keycloakRouteHost := printf "sso.%s" $ingressDomain }}
{{- $realmsName := "tssc-iam" }}
{{- $tpaTestingUsersEnabled := false }}
{{- $protocol := "https" -}}
{{- if $crc }}
  {{- $protocol = "http" }}
{{- end }}
{{- $tpaOIDCIssuerURL := printf "%s://%s/realms/%s" $protocol $keycloakRouteHost $realmsName }}

iam:
  enabled: {{ $keycloakEnabled }}
  namespace: {{ $keycloakNamespace }}
  instances: 1
  database:
    host: keycloak-pgsql
    name: keycloak
    secretName: keycloak-pgsql-user
  route:
    host: {{ $keycloakRouteHost }}
    tls:
      enabled: {{ not $crc }}
      secretName: {{ $keycloakRouteTLSSecretName }}
      termination: reencrypt
{{- if $crc }}
    annotations:
      route.openshift.io/termination: reencrypt
{{- end }}
  service:
    annotations:
      service.beta.openshift.io/serving-cert-secret-name: {{ $keycloakRouteTLSSecretName }}
  keycloakCR:
    namespace: {{ $keycloakNamespace }}
    ingressDomain: {{ $ingressDomain }}
    trustedArtifactSignerRealm:
      enabled: {{ $tas.Enabled }}
    trustedProfileAnalyzerRealm:
      enabled: {{ $tpa.Enabled }}
      oidcIssuerURL: {{ $tpaOIDCIssuerURL }}
      clients:
        cli:
          enabled: true
        testingManager:
          enabled: {{ $tpaTestingUsersEnabled }}
        testingUser:
          enabled: {{ $tpaTestingUsersEnabled }}
      frontendRedirectUris:
        - "http://localhost:8080"
    {{- range list "server" "sbom" }}
        - "{{ printf "%s://%s-%s.%s" $protocol . $tpa.Namespace $ingressDomain }}"
        - "{{ printf "%s://%s-%s.%s/*" $protocol . $tpa.Namespace $ingressDomain }}"
    {{- end }}
      integrationSecret:
        bombasticAPI: {{
          printf "%s://server-%s.%s"
            $protocol
            $tpa.Namespace
            $ingressDomain
        }}
        namespace: {{ .Installer.Namespace }}

#
# tssc-gitops
#
{{- $argoCDName := printf "%s-gitops" .Installer.Namespace }}

argoCD:
  enabled: {{ $gitops.Enabled }}
  name: {{ $argoCDName }}
  namespace: {{ $gitops.Namespace }}
  integrationSecret:
    name: tssc-argocd-integration
    namespace: {{ .Installer.Namespace }}
  ingressDomain: {{ $ingressDomain }}
  tssc:
    namespace: {{ .Installer.Namespace }}

#
# tssc-pipelines
#

pipelines:
  namespace: {{ $pipelinesNamespace }}
  tssc:
    namespace: {{ .Installer.Namespace }}

#
# tssc-integrations
#

integrations:
  argoCD:
    enabled: {{ $gitops.Enabled }}
    namespace: {{ $gitops.Namespace }}
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

#
# tssc-tpa
#

{{- $tpaDatabaseSecretName := "tpa-pgsql-user" }}
{{- $tpaAppDomain := printf "-%s.%s" $tpa.Namespace $ingressDomain }}
{{- $tpaOIDCClientsSecretName := "tpa-realm-clients" }}

trustedProfileAnalyzer:
  enabled: {{ $tpa.Enabled }}
  oidcIssuerURL: {{ $tpaOIDCIssuerURL }}
  namespace: "{{ $tpa.Namespace }}"
  appDomain: "{{ $tpaAppDomain }}"
  ingress: &tpaIngress
    className: openshift-default
  openshift: &tpaOpenShift
    # In practice it toggles "https" vs. "http" for TPA components, for CRC it's
    # easier to focus on "http" communication only.
    useServiceCa: {{ not $crc }}
  database: &tpaDatabase
    name:
      valueFrom:
        secretKeyRef:
          name: {{ $tpaDatabaseSecretName }}
          key: dbname
    host:
      valueFrom:
        secretKeyRef:
          name: {{ $tpaDatabaseSecretName }}
          key: host
    port:
      valueFrom:
        secretKeyRef:
          name: {{ $tpaDatabaseSecretName}}
          key: port
    username:
      valueFrom:
        secretKeyRef:
          name: {{ $tpaDatabaseSecretName }}
          key: user
    password:
      valueFrom:
        secretKeyRef:
          name: {{ $tpaDatabaseSecretName }}
          key: password
  createDatabase: *tpaDatabase
  migrateDatabase: *tpaDatabase
  storage: &tpaStorage
    type: filesystem
    size: 32Gi
  oidc: &tpaOIDC
    issuerUrl: {{ $tpaOIDCIssuerURL }}
    clients:
      cli:
        clientSecret:
          valueFrom:
            secretKeyRef:
              name: {{ $tpaOIDCClientsSecretName }}
              key: cli
{{- if $tpaTestingUsersEnabled }}
      testingUser:
        clientSecret:
          valueFrom:
            secretKeyRef:
              name: {{ $tpaOIDCClientsSecretName }}
              key: testingUser
      testingManager:
        clientSecret:
          valueFrom:
            secretKeyRef:
              name: {{ $tpaOIDCClientsSecretName }}
              key: testingManager
{{- end }}

trustification:
  name: trustedprofileanalyzer
  namespace: "{{ $tpa.Namespace }}"
  appDomain: "{{ $tpaAppDomain }}"
  openshift: *tpaOpenShift
  storage: *tpaStorage
  oidc: *tpaOIDC
  ingress: *tpaIngress
  tls:
    serviceEnabled: "{{ not $crc }}"

#
# tssc-tas
#

{{- $tasRealmPath := printf "realms/%s" $realmsName }}

trustedArtifactSigner:
  enabled: {{ $tas.Enabled }}
  ingressDomain: "{{ $ingressDomain }}"
  secureSign:
    enabled: {{ $tas.Enabled }}
    namespace: {{ $tas.Namespace }}
    fulcio:
      oidc:
        clientID: trusted-artifact-signer
        issuerURL: {{ printf "%s://%s/%s" $protocol $keycloakRouteHost $tasRealmPath }}
      certificate:
        # TODO: promopt the user for organization email/name input!
        organizationEmail: trusted-artifact-signer@company.dev
        organizationName: TSSC
  integrationSecret:
    namespace: {{ .Installer.Namespace }}
