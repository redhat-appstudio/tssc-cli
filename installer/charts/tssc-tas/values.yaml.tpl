{{- $crc := required "CRC settings" .Installer.Settings.crc -}}
{{- $tas := required "TAS settings" .Installer.Products.Trusted_Artifact_Signer -}}
{{- $ingressDomain := required "OpenShift ingress domain" .OpenShift.Ingress.Domain -}}
{{- $instNs := .Installer.Namespace -}}
{{- $tasActive := true -}}
{{- if and (kindIs "bool" $tas.Enabled) (eq $tas.Enabled false) }}{{- $tasActive = false -}}{{- end -}}
{{- $tasNs := default $instNs $tas.Namespace -}}
{{- $keycloakRouteHost := printf "tssc-sso.%s" $ingressDomain }}
{{- $realmsName := "tssc-iam" }}
{{- $protocol := "https" -}}
{{- if $crc }}
  {{- $protocol = "http" }}
{{- end }}
{{- $tasRealmPath := printf "realms/%s" $realmsName }}
---
trustedArtifactSigner:
  enabled: {{ $tasActive }}
  ingressDomain: "{{ $ingressDomain }}"
  secureSign:
    enabled: {{ $tasActive }}
    namespace: {{ $tasNs }}
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
