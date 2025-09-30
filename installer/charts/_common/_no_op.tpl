{{- define "common.noOp" -}}
#
# No op container
#
- name: no-op
    image: registry.redhat.io/openshift4/ose-tools-rhel9
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