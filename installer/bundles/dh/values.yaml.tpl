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
{{- $gitops := required "GitOps settings" .Installer.Products.OpenShift_GitOps -}}
{{- $pipelines := required "Pipelines settings" .Installer.Products.OpenShift_Pipelines -}}
{{- $pipelinesNamespace := "openshift-pipelines" -}}
{{- $tas := required "TAS settings" .Installer.Products.Trusted_Artifact_Signer -}}
{{- $instNs := .Installer.Namespace -}}
{{- $gitopsActive := true -}}
{{- if and (kindIs "bool" $gitops.Enabled) (eq $gitops.Enabled false) }}{{- $gitopsActive = false -}}{{- end -}}
{{- $gitopsNs := default $instNs $gitops.Namespace -}}
{{- $pipelinesActive := true -}}
{{- if and (kindIs "bool" $pipelines.Enabled) (eq $pipelines.Enabled false) }}{{- $pipelinesActive = false -}}{{- end -}}
{{- $tasActive := true -}}
{{- if and (kindIs "bool" $tas.Enabled) (eq $tas.Enabled false) }}{{- $tasActive = false -}}{{- end -}}
{{- $tasNs := default $instNs $tas.Namespace -}}
{{- $rhdhActive := true -}}
{{- if and (kindIs "bool" $rhdh.Enabled) (eq $rhdh.Enabled false) }}{{- $rhdhActive = false -}}{{- end -}}
{{- $rhdhNs := default $instNs $rhdh.Namespace -}}
{{- $rhdhRBAC := dict -}}
{{- $rhdhPropsDig := index $rhdh "Properties" | default dict -}}
{{- $rbacDig := index $rhdhPropsDig "RBAC" -}}
{{- if and $rbacDig (kindIs "map" $rbacDig) -}}
{{- $rhdhRBAC = $rbacDig -}}
{{- end -}}
#
# Namespace where tssc-iam creates rhdh-realm-clients (IAM bundle release ns). DH must
# lookup that Secret here — not only Installer.Namespace — or OIDC/Keycloak env keys are
# omitted when IAM runs in tssc-keycloak and integrations live in the installer project.
#
{{- $iamProd := index .Installer.Products "IAM" }}
{{- $iamKeycloakNs := "tssc-keycloak" }}
{{- if $iamProd }}
  {{- with index $iamProd "Namespace" }}
    {{- $iamKeycloakNs = . }}
  {{- end }}
{{- end }}
# openshift.projects for DH bundle (operator Subscriptions for GitOps/Pipelines/TAS live in their bundles).
openshift:
  projects:
    - {{ $gitopsNs }}
    - {{ $pipelinesNamespace }}
    - {{ $tasNs }}
    - {{ $rhdhNs }}
# tssc-dh-subscriptions only installs the Red Hat Developer Hub operator; other operators use tssc-*-subscriptions charts.
subscriptions:
  developerHub:
    enabled: {{ $rhdhActive }}
    managed: {{ and $rhdhActive $rhdh.Properties.manageSubscription }}
developerHub:
  namespace: {{ $rhdh.Namespace }}
  ingressDomain: {{ $ingressDomain }}
  catalogURL: {{ $catalogURL }}
  authProvider: {{ $authProvider }}
  keycloakIntegrationNamespace: {{ $iamKeycloakNs }}
  integrationSecrets:
    namespace: {{ .Installer.Namespace }}
  RBAC:
    enabled: {{ default false (index $rhdhRBAC "enabled") }}
{{- if eq $authProvider "github" }}
    adminUsers:
{{- $githubAdmins := list "${GITHUB__USERNAME}" -}}
{{- if hasKey $rhdhRBAC "adminUsers" }}{{- $githubAdmins = index $rhdhRBAC "adminUsers" -}}{{- end }}
{{ $githubAdmins | toYaml | indent 6 }}
    orgs:
{{- $githubOrgs := list "${GITHUB__ORG}" -}}
{{- if hasKey $rhdhRBAC "orgs" }}{{- $githubOrgs = index $rhdhRBAC "orgs" -}}{{- end }}
{{ $githubOrgs | toYaml | indent 6 }}
{{- else if eq $authProvider "gitlab" }}
    adminUsers:
{{- $gitlabAdmins := list "${GITLAB__USERNAME}" -}}
{{- if hasKey $rhdhRBAC "adminUsers" }}{{- $gitlabAdmins = index $rhdhRBAC "adminUsers" -}}{{- end }}
{{ $gitlabAdmins | toYaml | indent 6 }}
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
