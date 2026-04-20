{{/*
Expand the name of the chart.
*/}}
{{- define "tssc-quay.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create chart labels
*/}}
{{- define "tssc-quay.labels" -}}
helm.sh/chart: {{ include "tssc-quay.name" . }}
app.kubernetes.io/name: {{ include "tssc-quay.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Docker config JSON may arrive as a YAML object (map), JSON string, or []byte after config merge.
Emit a valid YAML scalar for stringData / env (Helm parses values with sigs.k8s.io/yaml).
*/}}
{{- define "tssc-quay.dockerconfigjsonQuoted" -}}
{{- $v := . -}}
{{- if kindIs "map" $v -}}
{{- $v | toJson | quote -}}
{{- else if kindIs "string" $v -}}
{{- if eq $v "" -}}
{{- "{}" | quote -}}
{{- else -}}
{{- $v | quote -}}
{{- end -}}
{{- else if kindIs "slice" $v -}}
{{- printf "%s" $v | quote -}}
{{- else if not $v -}}
{{- "{}" | quote -}}
{{- else -}}
{{- $v | toJson | quote -}}
{{- end -}}
{{- end -}}

{{- define "tssc-quay.dockerconfigjsonReadonlyQuoted" -}}
{{- $v := . -}}
{{- if kindIs "map" $v -}}
{{- $v | toJson | quote -}}
{{- else if kindIs "string" $v -}}
{{- $v | quote -}}
{{- else if kindIs "slice" $v -}}
{{- printf "%s" $v | quote -}}
{{- else if not $v -}}
{{- "" | quote -}}
{{- else -}}
{{- $v | toJson | quote -}}
{{- end -}}
{{- end -}}

{{/*
Scalar safe for YAML / Secret stringData (URLs with ports with colons, sprig quote(nil) is empty).
Empty/nil coerces to "" so stringData is never JSON null.
*/}}
{{- define "tssc-quay.scalarJSON" -}}
{{- if empty . -}}
""
{{- else -}}
{{- . | toJson -}}
{{- end -}}
{{- end -}}
