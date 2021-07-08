#!/usr/bin/env bash

DEFAULT_AWS_CLI_VERSION="2.0.52"

# daws() executes the AWS Python CLI tool from a Docker container.
#
# Instead of relying on developers having a particular version of the AWS
# Python CLI tool, this method allows a specific version of the CLI tool to be
# executed within a Docker container.
#
# You call the daws function just like you were calling the `aws` CLI tool.
#
# Usage:
#
#   daws SERVICE COMMAND [OPTIONS]
#
# Example:
#
#   daws ecr describe-repositories --repository-name my-repo
#
# To use a specific version of the AWS CLI, set the ACK_AWS_CLI_IMAGE_VERSION
# environment variable, otherwise the value of DEFAULT_AWS_CLI_VERSION is used.
daws() {
    aws_cli_profile_env=$([ ! -z "$AWS_PROFILE" ] && echo "--env AWS_PROFILE=$AWS_PROFILE")
    aws_cli_web_identity_env="$([ ! -z "$AWS_WEB_IDENTITY_TOKEN_FILE" ] && \
        echo "--env AWS_WEB_IDENTITY_TOKEN_FILE=/root/aws_token --env AWS_ROLE_ARN -v $AWS_WEB_IDENTITY_TOKEN_FILE:/root/aws_token:ro" )"
    aws_cli_img_version=${ACK_AWS_CLI_IMAGE_VERSION:-$DEFAULT_AWS_CLI_VERSION}
    aws_cli_img="amazon/aws-cli:$aws_cli_img_version"
    docker run --rm -v ~/.aws:/root/.aws:z $(echo $aws_cli_profile_env) $(echo $aws_cli_web_identity_env) -v $(pwd):/aws "$aws_cli_img" "$@"
}

# aws_check_credentials() calls the STS::GetCallerIdentity API call and
# verifies that there is a local identity for running AWS commands
aws_check_credentials() {
    echo -n "checking AWS credentials ... "
    daws sts get-caller-identity --query "Account" >/dev/null ||
        ( echo "\nFATAL: No AWS credentials found. Please run \`aws configure\` to set up the CLI for your credentials." && exit 1)
    echo "ok."
}

# generate_aws_temp_creds function will generate temporary AWS CREDENTIALS which are valid for 3600 seconds
aws_generate_temp_creds() {
    __uuid=$(uuidgen | cut -d'-' -f1 | tr '[:upper:]' '[:lower:]')

    if [ -z "$ACK_ROLE_ARN" ]; then
        printf "Missing input Role ARN, exiting...\n"
        exit 1
    fi

    JSON=$(daws sts assume-role \
           --role-arn "$ACK_ROLE_ARN"  \
           --role-session-name tmp-role-"$__uuid" \
           --duration-seconds 3600 \
           --output json || exit 1)

    AWS_ACCESS_KEY_ID=$(echo "${JSON}" | jq --raw-output ".Credentials[\"AccessKeyId\"]")
    AWS_SECRET_ACCESS_KEY=$(echo "${JSON}" | jq --raw-output ".Credentials[\"SecretAccessKey\"]")
    AWS_SESSION_TOKEN=$(echo "${JSON}" | jq --raw-output ".Credentials[\"SessionToken\"]")
}

aws_account_id() {
    JSON=$(daws sts get-caller-identity --output json || exit 1)
    echo "${JSON}" | jq --raw-output ".Account"
}

# eks_oidc_provider function provides OIDC Provider for given EKS Cluster.
# It accepts 1 argument:
#  - EKS_CLUSTER_NAME: name of the EKS Cluster.
eks_oidc_provider() {
    if [[ $# -ne 1 ]]; then
      echo "FATAL: Wrong number of arguments passed to ${FUNCNAME[0]}"
      echo "Usage: ${FUNCNAME[0]} EKS_CLUSTER_NAME"
      exit 1
    fi
    local EKS_CLUSTER_NAME="$1"
    local OIDC_PROVIDER_JSON=$(daws eks describe-cluster --name "$EKS_CLUSTER_NAME" --query "cluster.identity.oidc" --output json || exit 1 )
    local OIDC_PROVIDER=$(echo "${OIDC_PROVIDER_JSON}" | jq -re ".issuer // empty" | sed -e "s/^https:\/\///")
    echo "${OIDC_PROVIDER}"
}
