# BotKube

{{ template "chart.versionBadge" . }}{{ template "chart.typeBadge" . }}{{ template "chart.appVersionBadge" . }}

{{ template "chart.description" . }}

{{ template "chart.homepageLine" . }}

{{ define "chart.maintainersTable" }}
| Name | Email  |
| ---- | ------ |
{{- range .Maintainers }}
| {{ .Name }} | {{ if .Email }}<{{ .Email }}>{{ end }} |
{{- end }}
{{ end }}

{{ template "chart.maintainersSection" . }}

{{ template "chart.sourcesSection" . }}

{{ template "chart.requirementsSection" . }}

{{ define "chart.valuesHeader" }}## Parameters {{ end }}

{{ define "chart.valuesTable" }}
| Key | Type | Default | Description |
|-----|------|---------|-------------|
{{- range .Values }}
| [{{ .Key }}](./values.yaml#L{{ .LineNumber }}) | {{ .Type }} | {{ if .Default }}{{ .Default }}{{ else }}{{ .AutoDefault }}{{ end }} | {{ if .Description }}{{ regexReplaceAllLiteral "# .*" .Description }}{{ else }}{{ regexReplaceAllLiteral "# .*" .AutoDescription "" }}{{ end }} |
{{- end }}
{{- end }}

{{ template "chart.valuesSection" . }}

### AWS IRSA on EKS support

AWS has introduced IAM Role for Service Accounts in order to provide fine grained access. This is useful if you are looking to run BotKube inside an EKS cluster. For more details visit https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html.

Annotate the BotKube Service Account as shown in the example below and add the necessary Trust Relationship to the corresponding BotKube role to get this working.

```
serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: "<role_arn_to_assume>"
```
