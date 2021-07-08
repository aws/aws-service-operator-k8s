#!/usr/bin/env bash

# A script that setup IRSA (IAM roles for service accounts) on Amazon EKS cluster.

set -Eeo pipefail

SCRIPTS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

EKS_CLUSTER_NAME=${EKS_CLUSTER_NAME:-""}
AWS_REGION=${AWS_REGION:-"us-west-2"}
AWS_ACCOUNT_ID=${AWS_ACCOUNT_ID:-""}
ACK_K8S_NAMESPACE=${ACK_K8S_NAMESPACE:-"ack-system"}
ACK_K8S_SERVICE_ACCOUNT_NAME=${ACK_K8S_SERVICE_ACCOUNT_NAME:-""}
IAM_POLICY_ARN=${IAM_POLICY_ARN:-""}
OIDC_PROVIDER=${OIDC_PROVIDER:-""}


source "$SCRIPTS_DIR/lib/common.sh"
source "$SCRIPTS_DIR/lib/aws.sh"
ACK_LOG_LEVEL="debug"

check_is_installed docker
check_is_installed jq

USAGE="
Usage:
  export EKS_CLUSTER_NAME=\"\$CLUSTER_NAME\"
  export IAM_POLICY_ARN=\"\$AWS_IAM_POLICY_ARN\"
  $(basename "$0") <AWS_SERVICE>

Setup IRSA (IAM roles for service accounts) on Amazon EKS cluster.

Example: export EKS_CLUSTER_NAME=\"\$CLUSTER_NAME\"; export IAM_POLICY_ARN=\"\$AWS_IAM_POLICY_ARN\"; $(basename "$0") elasticache

<AWS_SERVICE> should be an AWS Service name (elasticache, ecr, sns, sqs, etc.)

Environment variables:
  EKS_CLUSTER_NAME:               Amazon EKS cluster name. Mandatory
  IAM_POLICY_ARN:                 ARN for IAM Policy. Mandatory
                                  Example for ElastiCache: 'arn:aws:iam::aws:policy/AmazonElastiCacheFullAccess'
  AWS_ACCOUNT_ID:                 AWS Account id.
                                  If not specified then it is retrieved using: aws sts get-caller-identity
  OIDC_PROVIDER:                  OIDC provider.
                                  If not specified then it is retrieved from EKS cluster details.
  AWS_REGION:                     AWS region. Default: us-west-2
  ACK_K8S_NAMESPACE:              ACK namespace.
                                  Default: ack-system
  ACK_K8S_SERVICE_ACCOUNT_NAME:   Service Account name.
                                  Default: ack-<AWS_SERVICE>-controller


For details on IRSA, refer https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html
"

if [ $# -ne 1 ]; then
    echo "AWS_SERVICE is not defined. Script accepts one parameter, <AWS_SERVICE>" 1>&2
    echo "${USAGE}"
    exit 1
fi

AWS_SERVICE=$(echo "$1" | tr '[:upper:]' '[:lower:]')

if [ -z "$ACK_K8S_SERVICE_ACCOUNT_NAME" ]; then
    ACK_K8S_SERVICE_ACCOUNT_NAME="ack-${AWS_SERVICE}-controller"
fi

if [ -z "$EKS_CLUSTER_NAME" ]; then
    echo "No Amazon EKS cluster name specified." 1>&2
    echo "${USAGE}"
    exit  1
fi

aws_check_credentials

if [ "z$AWS_ACCOUNT_ID" == "z" ]; then
    AWS_ACCOUNT_ID=$( aws_account_id )
fi

if [ -z "$AWS_ACCOUNT_ID" ]; then
    echo "No AWS Account Id found." 1>&2
    echo "${USAGE}"
    exit  1
fi

if [ "z$OIDC_PROVIDER" == "z" ]; then
    OIDC_PROVIDER=$( eks_oidc_provider "$EKS_CLUSTER_NAME" )
fi

if [ -z "$OIDC_PROVIDER" ]; then
    echo "No IAM OIDC provider is set for Amazon EKS cluster: $EKS_CLUSTER_NAME" 1>&2
    echo "${USAGE}"
    exit  1
fi

if [ -z "$IAM_POLICY_ARN" ]; then
    echo "No IAM Policy ARN specified" 1>&2
    echo "${USAGE}"
    exit  1
fi

export AWS_ACCOUNT_ID
export AWS_REGION

cat <<EOF > trust.json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::${AWS_ACCOUNT_ID}:oidc-provider/${OIDC_PROVIDER}"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "${OIDC_PROVIDER}:sub": "system:serviceaccount:${ACK_K8S_NAMESPACE}:${ACK_K8S_SERVICE_ACCOUNT_NAME}"
        }
      }
    }
  ]
}
EOF

ACK_CONTROLLER_IAM_ROLE="ack-${AWS_SERVICE}-controller"
ACK_CONTROLLER_IAM_ROLE_DESCRIPTION="IRSA role for ACK $AWS_SERVICE controller deployment on EKS cluster using Helm charts"

echo "setting up IRSA for cluster: $EKS_CLUSTER_NAME"
daws iam create-role --role-name "${ACK_CONTROLLER_IAM_ROLE}" --assume-role-policy-document file://trust.json --description "${ACK_CONTROLLER_IAM_ROLE_DESCRIPTION}"
daws iam attach-role-policy --role-name "${ACK_CONTROLLER_IAM_ROLE}" --policy-arn "$IAM_POLICY_ARN"
echo "done setting up IRSA for cluster $EKS_CLUSTER_NAME. Role Name: $ACK_CONTROLLER_IAM_ROLE"
