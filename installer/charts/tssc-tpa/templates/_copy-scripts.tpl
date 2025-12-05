{{/*

  POD container spec to copy scripts.

*/}}
{{- define "common.copyScripts" -}}
- name: copy-scripts
  image: registry.access.redhat.com/ubi9/ubi-minimal:latest
  workingDir: /scripts
  command:
    - /bin/bash
    - -c
    - |
      set -x -e
  {{- range $path, $content := .Files.Glob "scripts/*.sh" -}}
    {{- $script := trimPrefix "scripts/" $path }}
      printf '%s' "{{ $content | toString | b64enc }}" | base64 -d >{{ $script }}
      chmod +x {{ $script }}
  {{- end }}
  volumeMounts:
    - name: scripts
      mountPath: /scripts
  securityContext:
    allowPrivilegeEscalation: false
{{- end -}}
