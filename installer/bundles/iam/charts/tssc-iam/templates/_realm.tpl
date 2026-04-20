{{- define "generate.client.secret" -}}
{{-
  printf "%s-%s-%s-%s-%s"
    (randAlphaNum 8)
    (randAlphaNum 4)
    (randAlphaNum 4)
    (randAlphaNum 4)
    (randAlphaNum 12)
-}}
{{- end -}}
