# Setting up ACK with IAM Roles for Service Accounts

[IAM Roles for Service Accounts][0], or IRSA, is a system that automates the
provisioning and rotation of IAM temporary credentials (called a Web Identity)
that a Kubernetes `ServiceAccount` can use to call AWS APIs.

The primary advantage of IRSA is that Kubernetes `Pods` which use the
`ServiceAccount` associated with an IAM Role can have a reduced IAM permission
footprint than the IAM Role in use for the Kubernetes EC2 worker node (known as
the EC2 Instance Profile Role). This security concept is known as **Least
Privilege**.

For example, assume you have a broadly-scoped IAM Role with permissions to
access the Instance Metadata Service (IMDS) from the EC2 worker node. If you do
not want Kubernetes `Pods` running on that EC2 Instance to have access to IMDS,
you can create a different IAM Role with a reduced permission set and associate
this reduced-scope IAM Role with the Kubernetes `ServiceAccount` the `Pod`
uses. IRSA will ensure that a special file is injected (and rotated
periodically) into the `Pod` that contains a JSON Web Token (JWT) that
encapsulates a request for temporary credentials to assume the IAM Role with
reduced permissions.

When AWS clients or SDKs connect to an AWS API, they detect the existence of
this special token file and call the [`STS::AssumeRoleWithWebIdentity`][2] API
to assume the IAM Role with reduced permissions.

!!! note "EKS is not required to use IRSA"

    Note that you do *not* need to be using the Amazon EKS service in order to
    use IRSA. There are [instructions][1] on the
    amazon-eks-pod-identity-webhook repository for setting up IRSA on your own
    Kubernetes installation.

## IRSA setup on EKS cluster and install ACK controller using Helm
Following steps provide example to setup IRSA on EKS cluster to install ACK ElastiCache controller using Helm charts.
By modifying the variables values as needed, these steps can be applied for other ACK controllers.

The steps include:

### 1. Create OIDC identity provider for cluster
Create OIDC identity provider for cluster using CLI command.
Example:
```
EKS_CLUSTER_NAME=<eks cluster name>
eksctl utils associate-iam-oidc-provider --cluster $EKS_CLUSTER_NAME --approve
```
For detailed instructions, follow [Enabling IAM roles for service accounts on your cluster][3].

### 2. Create an IAM role and policy for service account
For detailed instructions, follow instructions at [Creating an IAM role and policy for your service account][4].

#### 2(a) - Create IAM role
```
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query "Account" --output text)
OIDC_PROVIDER=$(aws eks describe-cluster --name $EKS_CLUSTER_NAME --query "cluster.identity.oidc.issuer" --output text | sed -e "s/^https:\/\///")
ACK_K8S_NAMESPACE=ack-system
ACK_K8S_SERVICE_ACCOUNT_NAME=ack-elasticache-controller

read -r -d '' TRUST_RELATIONSHIP <<EOF
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
echo "${TRUST_RELATIONSHIP}" > trust.json

# update variables as needed
AWS_SERVICE_NAME='elasticache'
ACK_CONTROLLER_IAM_ROLE="ack-${AWS_SERVICE_NAME}-controller"
ACK_CONTROLLER_IAM_ROLE_DESCRIPTION='IRSA role for ACK $AWS_SERVICE_NAME controller deployment on EKS cluster using Helm charts'
aws iam create-role --role-name "${ACK_CONTROLLER_IAM_ROLE}" --assume-role-policy-document file://trust.json --description "${ACK_CONTROLLER_IAM_ROLE_DESCRIPTION}"
```

#### 2(b) - Attach IAM policy to role
```
# This example uses pre-existing policy for ElastiCache
# Create an IAM policy and use its ARN and update IAM_POLICY_ARN variable as needed
IAM_POLICY_ARN='arn:aws:iam::aws:policy/AmazonElastiCacheFullAccess'
aws iam attach-role-policy \
    --role-name "${ACK_CONTROLLER_IAM_ROLE}" \
    --policy-arn "$IAM_POLICY_ARN"
```

### 3. Associate an IAM role to service account

For detailed instructions, follow instructions at [Associate an IAM role to a service account ][5].

#### 3(a) - If Helm charts available on local file system

Update `values.yaml` and set value for `aws.region`, `serviceAccount.annotations`.

```
# update variables as needed
ACK_CONTROLLER_HELM_CHARTS_DIR=<directory containing Helm chart for ACK service controller>
AWS_SERVICE_NAME='elasticache'
ACK_K8S_NAMESPACE=ack-system
ACK_K8S_RELEASE_NAME=ack-$AWS_SERVICE_NAME-controller

kubectl create namespace "$ACK_K8S_NAMESPACE"
cd "$ACK_CONTROLLER_HELM_CHARTS_DIR"

# dry run and view the resultant output
helm install --debug --dry-run --namespace "$ACK_K8S_NAMESPACE" "$ACK_K8S_RELEASE_NAME" .
# install on cluster
helm install --namespace "$ACK_K8S_NAMESPACE" "$ACK_K8S_RELEASE_NAME" .
```

Verify that the service account has been created on cluster and that its annotation include IAM Role
 (created during Step#2 above) arn:
```
kubectl describe serviceaccount/$ACK_K8S_SERVICE_ACCOUNT_NAME -n $ACK_K8S_NAMESPACE
```

#### 3(b) - If Helm charts have already been installed on cluster without modifying `values.yaml`

For example, if installation was done as:
```
AWS_SERVICE_NAME='elasticache'
ACK_K8S_NAMESPACE=ack-system
ACK_K8S_RELEASE_NAME=ack-$AWS_SERVICE_NAME-controller
helm install --namespace $ACK_K8S_NAMESPACE ack-$AWS_SERVICE_NAME-controller $ACK_K8S_RELEASE_NAME
```
Then service account would already exist on the cluster; however its association with IAM Role would be pending.
Verify it using:
```
kubectl describe serviceaccount/$ACK_K8S_SERVICE_ACCOUNT_NAME -n $ACK_K8S_NAMESPACE
```
Observe that the arn of IAM Role (created during Step#2 above) is not set as annotation for the service account.

To associate an IAM role to service account:
```
# annotate service account with service role arn.
ISRA_ROLE_ARN=<role arn>
kubectl annotate serviceaccount -n $ACK_K8S_NAMESPACE $ACK_K8S_SERVICE_ACCOUNT_NAME eks.amazonaws.com/role-arn=$ISRA_ROLE_ARN
```

Update aws region to use in the controller, if not done already:
```
# update desired AWS region. example: us-east-1
AWS_REGION=<aws region id>
kubectl -n $ACK_K8S_NAMESPACE set env deployment/$ACK_K8S_RELEASE_NAME \
    AWS_REGION="$AWS_ACCOUNT_ID"
```

### Verify
Describe one of the pods and verify that the `AWS_WEB_IDENTITY_TOKEN_FILE` and `AWS_ROLE_ARN` environment variables exist.
```
kubectl get pods -A
kubectl exec -n kube-system aws-node-<9rgzw> env | grep AWS
```
verify the output, example:
```
AWS_VPC_K8S_CNI_LOGLEVEL=DEBUG
AWS_ROLE_ARN=arn:aws:iam::<AWS_ACCOUNT_ID>:role/<IAM_ROLE_NAME>
AWS_WEB_IDENTITY_TOKEN_FILE=/var/run/secrets/eks.amazonaws.com/serviceaccount/token
```

## 

[0]: https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html
[1]: https://github.com/aws/amazon-eks-pod-identity-webhook/blob/master/SELF_HOSTED_SETUP.md
[2]: https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRoleWithWebIdentity.html
[3]: https://docs.aws.amazon.com/eks/latest/userguide/enable-iam-roles-for-service-accounts.html
[4]: https://docs.aws.amazon.com/eks/latest/userguide/create-service-account-iam-policy-and-role.html
[5]: https://docs.aws.amazon.com/eks/latest/userguide/specify-service-account-role.html
[6]: https://docs.aws.amazon.com/eks/latest/userguide/eksctl.html#installing-eksctl