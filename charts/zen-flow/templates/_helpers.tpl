{{/*
Expand the name of the chart.
*/}}
{{- define "zen-flow.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "zen-flow.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "zen-flow.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "zen-flow.labels" -}}
helm.sh/chart: {{ include "zen-flow.chart" . }}
{{ include "zen-flow.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "zen-flow.selectorLabels" -}}
app.kubernetes.io/name: {{ include "zen-flow.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "zen-flow.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "zen-flow.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Namespace to use
*/}}
{{- define "zen-flow.namespace" -}}
{{- default .Values.namespace.name .Release.Namespace }}
{{- end }}

{{/*
Leader election namespace
*/}}
{{- define "zen-flow.leaderElectionNamespace" -}}
{{- if .Values.leaderElection.namespace }}
{{- .Values.leaderElection.namespace }}
{{- else }}
{{- include "zen-flow.namespace" . }}
{{- end }}
{{- end }}

