# Default values for ack-{{ .ServiceIDClean }}-controller.
# This is a YAML-formatted file.
# Declare variables to be passed into your templates.

image:
  repository: {{ .ImageRepository }}
  tag: {{ .ServiceIDClean }}-{{ .ReleaseVersion }}
  pullPolicy: IfNotPresent
  pullSecrets: []

nameOverride: ""
fullnameOverride: ""

deployment:
  annotations: {}
  labels: {}

resources:
  requests:
    memory: "64Mi"
    cpu: "50m"
  limits:
    memory: "128Mi"
    cpu: "100m"

aws:
  # If specified, use the AWS region for AWS API calls
  region: ""
  credentialsSecretName: ""

serviceAccount:
  # Specifies whether a service account should be created
  create: true
  # The name of the service account to use.
  name: {{ .ServiceAccountName }}
  annotations: {}
    # eks.amazonaws.com/role-arn: arn:aws:iam::AWS_ACCOUNT_ID:role/IAM_ROLE_NAME
