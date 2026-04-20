{{- $crc := required "CRC settings" .Installer.Settings.crc -}}
{{- $tas := required "TAS settings" .Installer.Products.Trusted_Artifact_Signer -}}
{{- $rhdh := required "RHDH settings" .Installer.Products.Developer_Hub -}}
{{- $ingressDomain := required "OpenShift ingress domain" .OpenShift.Ingress.Domain -}}
{{- $authProvider := required "Auth Provider is required" $rhdh.Properties.authProvider }}
{{- $instNs := .Installer.Namespace -}}
{{- $tasActive := true -}}
{{- if and (kindIs "bool" $tas.Enabled) (eq $tas.Enabled false) }}{{- $tasActive = false -}}{{- end -}}
{{- $rhdhActive := true -}}
{{- if and (kindIs "bool" $rhdh.Enabled) (eq $rhdh.Enabled false) }}{{- $rhdhActive = false -}}{{- end -}}
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
{{- $tpaRealmEnabled := or $tpaActive (not (empty $tpaIntegration)) -}}
{{- $keycloakEnabled := or $tpaActive $tasActive (and $rhdhActive (eq $authProvider "oidc")) (not (empty $tpaIntegration)) }}
{{- $keycloakNamespace := "tssc-keycloak" -}}
{{- $keycloakRouteTLSSecretName := "keycloak-tls" }}
{{- $keycloakRouteHost := printf "tssc-sso.%s" $ingressDomain }}
{{- $realmsName := "tssc-iam" }}
{{- $tpaTestingUsersEnabled := false }}
{{- $protocol := "https" -}}
{{- if $crc }}
  {{- $protocol = "http" }}
{{- end }}
{{- $tpaOIDCIssuerURL := printf "%s://%s/realms/%s" $protocol $keycloakRouteHost $realmsName }}
---
iam:
  enabled: {{ $keycloakEnabled }}
  namespace: {{ $keycloakNamespace }}
  integrationSecret:
    namespace: {{ .Installer.Namespace }}
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
    rhdhRealm:
      enabled: {{ and $rhdhActive (eq $authProvider "oidc") }}
      rhdhRedirectUris:
        - {{
          printf "%s://backstage-developer-hub-tssc-dh.%s/api/auth/oidc/handler/frame"
          $protocol
          $ingressDomain
        }}
      rhdhOriginUris:
        - {{
          printf "%s://backstage-developer-hub-tssc-dh.%s"
          $protocol
          $ingressDomain
        }}
    trustedArtifactSignerRealm:
      enabled: {{ $tasActive }}
    trustedProfileAnalyzerRealm:
      enabled: {{ $tpaRealmEnabled }}
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
        - "{{ printf "%s://%s-%s.%s" $protocol . $tpaNs $ingressDomain }}"
        - "{{ printf "%s://%s-%s.%s/*" $protocol . $tpaNs $ingressDomain }}"
    {{- end }}
