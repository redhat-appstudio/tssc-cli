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
  serviceAccountName: {{ .serviceAccount | default .Release.Name }}
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