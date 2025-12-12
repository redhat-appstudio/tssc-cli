{{- define "common.noOp" -}}
#
# No op container
#
- name: no-op
    image: registry.access.redhat.com/ubi10/ubi-minimal:latest
    command:
        - bash
        - -c
        - "echo 'No op: Success'"
    requests:
        cpu: 125m
        memory: 128Mi
        ephemeral-storage: "100Mi"
    securityContext:
        runAsNonRoot: false
        allowPrivilegeEscalation: false
{{- end }}