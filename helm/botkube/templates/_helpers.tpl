{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "botkube.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "botkube.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "botkube.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "botkube.serviceAccountName" -}}
{{- if .Values.serviceAccount.name -}}
    {{ .Values.serviceAccount.name }}
{{- else -}}
    {{ include "botkube.fullname" . }}-sa
{{- end -}}
{{- end -}}

{{- define "botkube.CommunicationsSecretName" -}}
{{- .Values.existingCommunicationsSecretName | default (printf "%s-communication-secret" (include "botkube.fullname" .)) -}}
{{- end -}}

{{- define "botkube.SSLCertSecretName" -}}
{{- .Values.ssl.existingSecretName | default (printf "%s-certificate-secret" (include "botkube.fullname" .)) -}}
{{- end -}}

{{- define "botkube.communication.team.enabled" -}}
{{- range $key, $val := .Values.communications -}}
{{- if $val.teams.enabled -}}
  {{- true -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "botkube.remoteConfigEnabled" -}}
{{ if .Values.config.provider.identifier }}
    {{- true -}}
{{- end -}}
{{- end -}}

{{- define "botkube.uuid" -}}
{{- $uuid := printf "%s-%s-%s-%s-%s" (randAlphaNum 8) (randAlphaNum 4) "4" (randAlphaNum 3) (randAlphaNum 12) -}}
{{- $uuid | lower -}}
{{- end -}}
