{{- with (index .Installer.Integrations "quay") -}}
{{- $p := .Properties | default dict -}}
{{- $dcj := index $p "dockerconfigjson" -}}
{{- $dcjr := index $p "dockerconfigjsonreadonly" -}}
---
quay:
  url: {{ index $p "url" | toJson }}
  token: {{ index $p "token" | toJson }}
  organization: {{ index $p "organization" | toJson }}
  dockerconfigjson: {{ if kindIs "map" $dcj }}{{ $dcj | toJson | quote }}{{ else if kindIs "string" $dcj }}{{ $dcj | quote }}{{ else if kindIs "slice" $dcj }}{{ printf "%s" $dcj | quote }}{{ else }}{{ "{}" | quote }}{{ end }}
  dockerconfigjsonreadonly: {{ if kindIs "map" $dcjr }}{{ $dcjr | toJson | quote }}{{ else if kindIs "string" $dcjr }}{{ $dcjr | quote }}{{ else if kindIs "slice" $dcjr }}{{ printf "%s" $dcjr | quote }}{{ else }}{{ "" | quote }}{{ end }}
{{- else -}}
---
quay:
  url: ""
  token: ""
  organization: ""
  dockerconfigjson: "{}"
  dockerconfigjsonreadonly: ""
{{- end }}
