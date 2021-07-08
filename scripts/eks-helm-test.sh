#!/usr/bin/env bash

# A script that installs specified ACK service controller helm charts to Amazon EKS cluster
# and runs e2e tests

set -Eeo pipefail

SCRIPTS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
ROOT_DIR="$SCRIPTS_DIR/.."
TEST_DIR="$ROOT_DIR/test"
TEST_E2E_DIR="$TEST_DIR/e2e"

EKS_CLUSTER_NAME=${EKS_CLUSTER_NAME:-""}
ACK_CONTROLLER_RELEASE_VERSION=${ACK_CONTROLLER_RELEASE_VERSION:-""}
IRSA_ROLE_ARN=${IRSA_ROLE_ARN:-""}
ACK_K8S_SERVICE_ACCOUNT_NAME=${ACK_K8S_SERVICE_ACCOUNT_NAME:-""}

AWS_REGION=${AWS_REGION:-"us-west-2"}
ACK_K8S_NAMESPACE=${ACK_K8S_NAMESPACE:-"ack-system"}
AWS_ACCOUNT_ID=${AWS_ACCOUNT_ID:-""}
SKIP_PYTHON_TESTS=${SKIP_PYTHON_TESTS:-"false"}
RUN_PYTEST_LOCALLY=${RUN_PYTEST_LOCALLY:="false"}
ACK_LOG_LEVEL="debug"

DEFAULT_HELM_REGISTRY="public.ecr.aws/aws-controllers-k8s"
DEFAULT_HELM_REPO="chart"

HELM_REGISTRY=${HELM_REGISTRY:-$DEFAULT_HELM_REGISTRY}
HELM_REPO=${HELM_REPO:-$DEFAULT_HELM_REPO}


source "$SCRIPTS_DIR/lib/common.sh"
source "$SCRIPTS_DIR/lib/aws.sh"
source "$SCRIPTS_DIR/lib/k8s.sh"

check_is_installed docker
check_is_installed jq
check_is_installed uuidgen
check_is_installed helm "You can install Helm with the helper scripts/install-helm.sh"
check_is_installed kubectl "You can install kubectl with the helper scripts/install-kubectl.sh"

aws_check_credentials

if [ "z$AWS_ACCOUNT_ID" == "z" ]; then
    AWS_ACCOUNT_ID=$( aws_account_id )
fi

USAGE="
Usage:
  export EKS_CLUSTER_NAME=\"\$CLUSTER_NAME\"
  export ACK_CONTROLLER_RELEASE_VERSION=\"\$chart_version\"
  export IRSA_ROLE_ARN=\"\$irsa role arn>\"
  $(basename "$0") <AWS_SERVICE>

Installs specified ACK service controller helm charts to Amazon EKS cluster
and runs e2e tests

Example: export EKS_CLUSTER_NAME=\"\$CLUSTER_NAME\"; export ACK_CONTROLLER_RELEASE_VERSION=\"\$chart_version\"; export IRSA_ROLE_ARN=\"\$irsa role arn>\" $(basename "$0") elasticache

<AWS_SERVICE> should be an AWS Service name (ecr, sns, sqs, petstore, bookstore)

Environment variables:
  EKS_CLUSTER_NAME:                   Amazon EKS cluster name. Mandatory
  ACK_CONTROLLER_RELEASE_VERSION:     The semver release version to use. Mandatory
                                      Example: v0.0.3
  IRSA_ROLE_ARN:                      IRSA Role ARN. Mandatory.
                                      Example: IRSA_ROLE_ARN=arn:aws:iam::<AWS_ACCOUNT_ID>:role/<IAM_ROLE_NAME>
  ACK_K8S_SERVICE_ACCOUNT_NAME:       Service Account name.
                                      Default: ack-<AWS_SERVICE>-controller
  AWS_REGION:                         AWS region.
                                      Default: us-west-2
  HELM_REGISTRY:                      The name of the Helm registry.
                                      Default: $DEFAULT_HELM_REGISTRY
  HELM_REPO:                          The name of the Helm repository.
                                      Default: $DEFAULT_HELM_REPO
  ACK_K8S_NAMESPACE:                  ACK namespace.
                                      Default: ack-system
  SKIP_PYTHON_TESTS:                  Whether to skip python tests and run bash tests instead for
                                      the service controller (<true|false>)
                                      Default: false
  RUN_PYTEST_LOCALLY:                 If python tests exist, whether to run them locally instead of
                                      inside Docker (<true|false>)
                                      Default: false
"

if [ $# -ne 1 ]; then
    echo "AWS_SERVICE is not defined. Script accepts one parameter, <AWS_SERVICE>" 1>&2
    echo "${USAGE}"
    exit 1
fi

AWS_SERVICE=$(echo "$1" | tr '[:upper:]' '[:lower:]')

if [ -z "$EKS_CLUSTER_NAME" ]; then
    echo "No Amazon EKS cluster name specified." 1>&2
    echo "${USAGE}"
    exit  1
fi

if [ -z "$EKS_CLUSTER_NAME" ]; then
    echo "No Amazon EKS cluster name specified." 1>&2
    echo "${USAGE}"
    exit  1
fi

if [ -z "$ACK_CONTROLLER_RELEASE_VERSION" ]; then
    echo "No release version specified for ACK $AWS_SERVICE controller." 1>&2
    echo "${USAGE}"
    exit  1
fi

if [ -z "$IRSA_ROLE_ARN" ]; then
    echo "No IRSA Role ARN specified to setup service account for ACK $AWS_SERVICE controller." 1>&2
    echo "${USAGE}"
    exit  1
fi

if [ -z "$ACK_K8S_SERVICE_ACCOUNT_NAME" ]; then
    ACK_K8S_SERVICE_ACCOUNT_NAME="ack-${AWS_SERVICE}-controller"
fi

if [ -z "$TMP_DIR" ]; then
    TEST_ID=$(uuidgen | cut -d'-' -f1 | tr '[:upper:]' '[:lower:]')
    TMP_DIR=$ROOT_DIR/build/tmp-eks-test-$TEST_ID
fi

echo "To run the tests, using temporary directory: $TMP_DIR"
mkdir -p "$TMP_DIR"

KUBECONFIG_PATH="${TMP_DIR}/kubeconfig"
daws eks update-kubeconfig --name "$EKS_CLUSTER_NAME" --kubeconfig "$KUBECONFIG_PATH"
if [[ ! -f "$KUBECONFIG_PATH" ]]; then
    echo "Failed to setup kubeconfig for Amazon EKS cluster: $EKS_CLUSTER_NAME" 1>&2
    echo "${USAGE}"
    exit 1
fi

KUBECONFIG="$KUBECONFIG_PATH"
export KUBECONFIG

echo "======================================================================================================"
echo "To poke around your test cluster manually:"
echo "export KUBECONFIG=$KUBECONFIG_PATH"
echo "kubectl get pods -A"
echo "======================================================================================================"

# install helm chart
ACK_K8S_RELEASE_NAME="ack-${AWS_SERVICE}-controller"
REMOTE_HELM_CHARTS="$HELM_REGISTRY/$HELM_REPO:${AWS_SERVICE}-$ACK_CONTROLLER_RELEASE_VERSION"

HELM_EXPERIMENTAL_OCI=1
export HELM_EXPERIMENTAL_OCI

echo "Installing helm charts: $REMOTE_HELM_CHARTS on Amazon EKS cluster: $EKS_CLUSTER_NAME"
echo "Helm chart install dryrun output location: ${TMP_DIR}/helm_install_dryrun_output"

LOCAL_HELM_CHARTS="${TMP_DIR}/charts"

helm chart pull "$REMOTE_HELM_CHARTS"
helm chart export "$REMOTE_HELM_CHARTS" --destination "$LOCAL_HELM_CHARTS"

LOCAL_SERVICE_HELM_CHART="$LOCAL_HELM_CHARTS/ack-${AWS_SERVICE}-controller/"

echo "Dowloaded charts at: $LOCAL_SERVICE_HELM_CHART"

helm install --debug --dry-run --namespace "$ACK_K8S_NAMESPACE" "$ACK_K8S_RELEASE_NAME" "$LOCAL_SERVICE_HELM_CHART" > "${TMP_DIR}/helm_install_dryrun_output"

echo "Proceeding with chart install"

NAMESPACE_ON_CLUSTER=$(kubectl get namespace -o json | jq -re ".items[].metadata" | jq -r "select(.name == \"$ACK_K8S_NAMESPACE\")" | jq -re ".name")

if [ -z "$NAMESPACE_ON_CLUSTER" ]; then
    echo "Namespace: $ACK_K8S_NAMESPACE does not exist on Amazon EKS cluster: $EKS_CLUSTER_NAME. Creating it."
    kubectl create namespace "$ACK_K8S_NAMESPACE"
fi

helm install --namespace "$ACK_K8S_NAMESPACE" "$ACK_K8S_RELEASE_NAME" "$LOCAL_SERVICE_HELM_CHART"

sleep 60

echo "Annotating Service Account with service Role ARN."
kubectl annotate serviceaccount -n "$ACK_K8S_NAMESPACE" $ACK_K8S_SERVICE_ACCOUNT_NAME eks.amazonaws.com/role-arn="$IRSA_ROLE_ARN"

sleep 60

echo "Setting deployment environment variable AWS_REGION to: $AWS_REGION"
kubectl -n "$ACK_K8S_NAMESPACE" set env "deployment/$ACK_K8S_RELEASE_NAME" AWS_REGION="$AWS_REGION"

sleep 60

echo "Starting e2e tests"
# run e2e tests
export SKIP_PYTHON_TESTS
export RUN_PYTEST_LOCALLY

$TEST_E2E_DIR/run-tests.sh "$AWS_SERVICE"