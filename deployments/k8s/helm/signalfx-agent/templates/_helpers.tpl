{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "signalfx-agent.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "signalfx-agent.fullname" -}}
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
{{- define "signalfx-agent.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "signalfx-agent.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "signalfx-agent.fullname" .) .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Get namespace to deploy agent and its dependencies.
*/}}
{{- define "signalfx-agent.namespace" -}}
    {{- default .Release.Namespace .Values.namespace -}}
{{- end -}}

{{/*
Create the name of the secret holding the token
*/}}
{{- define "signalfx-agent.secretName" -}}
    {{ default (include "signalfx-agent.fullname" .) .Values.signalFxAccessTokenSecretName }}
{{- end -}}
{{/*
Name of secret holding splunk hec token
*/}}
{{- define "signalfx-agent.secretNameSplunk" -}}
    {{ default (include "signalfx-agent.secretName" .) .Values.splunkTokenSecretName }}
{{- end -}}


{{/*
Create the configmap name. It will have -v<major version> appended to it from v5 and onward.
*/}}
{{- define "configmap-name" -}}
{{- $major_version := .Values.agentVersion | default "0.0.0" | splitList "." | first | atoi -}}
{{- if ge $major_version 5 -}}
{{- template "signalfx-agent.fullname" . -}}-v{{- $major_version -}}
{{- else -}}
{{- template "signalfx-agent.fullname" . -}}
{{- end -}}
{{- end -}}
