{{/*
  Not symlinked to charts/_common/_test.tpl: tests/test.yaml passes the tssc-dh-access
  ServiceAccount rhdh-kubernetes-plugin (ClusterRole read access to deployments cluster-wide).
  Pre-bundle installer/charts/tssc-dh used the same SA so rollout-status worked even when the
  hook Pod ran in developerHub.integrationSecrets.namespace; pod/sa namespace must match — use
  .Release.Namespace with rhdh-kubernetes-plugin unless .serviceAccount is set explicitly.
*/}}
{{- define "common.test" -}}
apiVersion: v1
kind: Pod
metadata:
  annotations:
    helm.sh/hook: test
    helm.sh/hook-delete-policy: before-hook-creation,hook-succeeded
  labels:
    {{- include "common.labels" . | nindent 4 }}
  name: test-{{ .name | default .Chart.Name }}
  namespace: {{ .namespace | default .Release.Namespace }}
spec:
  restartPolicy: Never
  serviceAccountName: {{ .serviceAccount | default "rhdh-kubernetes-plugin" }}
  initContainers:
    #
    # Copying the scripts that will be used on the subsequent containers, the
    # scripts are shared via the "/scripts" volume.
    #
{{- include "common.copyScripts" . | nindent 4 }}
  volumes:
    - name: scripts
      emptyDir: {}
{{- end -}}
