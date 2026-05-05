{{- $crc := required "CRC settings" .Installer.Settings.crc -}}
{{- $rhdh := required "RHDH settings" .Installer.Products.Developer_Hub -}}
{{- $ingressDomain := required "OpenShift ingress domain" .OpenShift.Ingress.Domain -}}
{{- $authProvider := required "Auth Provider is required" $rhdh.Properties.authProvider }}
{{- $keycloakRouteHost := printf "tssc-sso.%s" $ingressDomain }}
{{- $realmsName := "tssc-iam" }}
{{- $protocol := "https" -}}
{{- if $crc }}
  {{- $protocol = "http" }}
{{- end }}
{{- $catalogURL := required "Red Hat Developer Hub Catalog URL is required"
    $rhdh.Properties.catalogURL }}
---
developerHub:
  namespace: {{ $rhdh.Namespace }}
  ingressDomain: {{ $ingressDomain }}
  catalogURL: {{ $catalogURL }}
  authProvider: {{ $authProvider }}
  integrationSecrets:
    namespace: {{ .Installer.Namespace }}
  RBAC:
    enabled: {{ dig "Properties" "RBAC" "enabled" false $rhdh }}
{{- if eq $authProvider "github" }}
    adminUsers:
{{ dig "Properties" "RBAC" "adminUsers" (list "${GITHUB__USERNAME}") $rhdh | toYaml | indent 6 }}
    orgs:
{{ dig "Properties" "RBAC" "orgs" (list "${GITHUB__ORG}") $rhdh | toYaml | indent 6 }}
{{- else if eq $authProvider "gitlab" }}
    adminUsers:
{{ dig "Properties" "RBAC" "adminUsers" (list "${GITLAB__USERNAME}") $rhdh | toYaml | indent 6 }}
{{- else if eq $authProvider "oidc" }}
  oidc:
    secretNamespace: {{ .Installer.Namespace }}
    enabled: true
    clientID: rhdh
    metadataURL: {{ printf "%s://%s/realms/%s/.well-known/openid-configuration" $protocol $keycloakRouteHost $realmsName }}
    baseURL: {{ printf "%s://%s" $protocol $keycloakRouteHost }}
    loginRealm: {{ $realmsName }}
    realm: {{ $realmsName }}
{{- end }}
