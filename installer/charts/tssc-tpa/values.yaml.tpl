{{- $crc := required "CRC settings" .Installer.Settings.crc -}}
{{- $instNs := .Installer.Namespace -}}
{{- $ingressDomain := required "OpenShift ingress domain" .OpenShift.Ingress.Domain -}}
{{- $keycloakRouteHost := printf "tssc-sso.%s" $ingressDomain }}
{{- $realmsName := "tssc-iam" }}
{{- $protocol := "https" -}}
{{- if $crc }}{{- $protocol = "http" -}}{{- end -}}
{{- $tpaOIDCIssuerURL := printf "%s://%s/realms/%s" $protocol $keycloakRouteHost $realmsName }}
{{- $tpaDatabaseSecretName := "tpa-pgsql-user" }}
{{- $tpaOIDCClientsSecretName := "tpa-realm-clients" }}
{{- $tpaTestingUsersEnabled := false }}
{{- $tpa := index .Installer.Products "Trusted_Profile_Analyzer" -}}
{{- $bundleMode := "product" -}}
{{- if not $tpa }}{{- $bundleMode = "integration" -}}{{- end -}}
{{- if $tpa -}}
{{- if and (kindIs "bool" $tpa.Enabled) (eq $tpa.Enabled false) }}{{- $bundleMode = "integration" -}}{{- end -}}
{{- end -}}
{{- $tpaNs := "tssc-tpa" -}}
{{- if eq $bundleMode "product" -}}
{{- $tpaNs = default $instNs $tpa.Namespace -}}
{{- else if $tpa -}}
{{- $tpaNs = default "tssc-tpa" $tpa.Namespace -}}
{{- end -}}
{{- $tpaAppDomain := printf "-%s.%s" $tpaNs $ingressDomain -}}
{{- $trustIn := index .Installer.Integrations "tpa" -}}
{{- $trustProps := dict -}}
{{- if $trustIn }}{{- $trustProps = index $trustIn "Properties" | default dict -}}{{- end -}}
{{- $defaultBombastic := printf "%s://server%s" $protocol $tpaAppDomain -}}
{{- $bombasticEffective := $defaultBombastic -}}
{{- with index $trustProps "bombastic_api_url" -}}
{{- if . }}{{- $bombasticEffective = toString . -}}{{- end -}}
{{- end -}}
{{- $cycloneEffective := "1.4" -}}
{{- if and $tpa (index $tpa "cycloneDXVersion") }}{{- $cycloneEffective = toString (index $tpa "cycloneDXVersion") -}}{{- end -}}
{{- with index $trustProps "supported_cyclonedx_version" -}}
{{- if ne (toString .) "" }}{{- $cycloneEffective = toString . -}}{{- end -}}
{{- end -}}
{{- if eq $bundleMode "integration" -}}
---
bundle:
  mode: integration
trustedProfileAnalyzer:
  enabled: false
  cycloneDXVersion: "1.4"
  oidcIssuerURL: {{ $tpaOIDCIssuerURL }}
  namespace: "{{ $tpaNs }}"
  appDomain: "{{ $tpaAppDomain }}"
  integrationSecret:
    namespace: {{ $instNs }}
    bombastic_api_url: {{ $bombasticEffective | quote }}
    supported_cyclonedx_version: {{ $cycloneEffective | quote }}
  openshift: &tpaOpenShift
    useServiceCa: {{ not $crc }}
trustification:
  name: trustedprofileanalyzer
  namespace: "{{ $tpaNs }}"
  appDomain: "{{ $tpaAppDomain }}"
  openshift: *tpaOpenShift
  oidc:
    issuerUrl: {{ $tpaOIDCIssuerURL }}
  ingress:
    className: openshift-default
  tls:
    serviceEnabled: "{{ not $crc }}"
{{- else -}}
{{- $_ := required "TPA settings" $tpa -}}
{{- $tpaActive := true -}}
{{- if and (kindIs "bool" $tpa.Enabled) (eq $tpa.Enabled false) }}{{- $tpaActive = false -}}{{- end -}}
---
bundle:
  mode: product
trustedProfileAnalyzer:
  enabled: {{ $tpaActive }}
  cycloneDXVersion: "1.4"
  oidcIssuerURL: {{ $tpaOIDCIssuerURL }}
  namespace: "{{ $tpaNs }}"
  appDomain: "{{ $tpaAppDomain }}"
  integrationSecret:
    namespace: {{ $instNs }}
    bombastic_api_url: {{ $bombasticEffective | quote }}
    supported_cyclonedx_version: {{ $cycloneEffective | quote }}
  ingress: &tpaIngress
    className: openshift-default
  openshift: &tpaOpenShift
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
  namespace: "{{ $tpaNs }}"
  appDomain: "{{ $tpaAppDomain }}"
  openshift: *tpaOpenShift
  storage: *tpaStorage
  oidc: *tpaOIDC
  ingress: *tpaIngress
  tls:
    serviceEnabled: "{{ not $crc }}"
{{- end }}
