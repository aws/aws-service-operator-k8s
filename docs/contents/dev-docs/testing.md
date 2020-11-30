# Testing

Welcome to the ACK developer preview! In the following, we will take you
through the steps to test ACK for the currently supported AWS services:

- Amazon ECR
- Amazon S3
- Amazon SNS
* Amazon SQS
- Amazon API Gateway V2
- Amazon DynamoDB
- [Amazon ElastiCache](https://github.com/aws/aws-controllers-k8s/tree/main/services/elasticache)
- AWS Step Functions

If you run into any problems when testing one of the above services,
[raise an issue](https://github.com/aws/aws-controllers-k8s/issues/new/choose)
with the details so we can reproduce your issue.

## Prerequisites

For local development and testing we use "Kubernetes in Docker" (`kind`), 
which in turn requires Docker.

!!! warning "Footprint"
    When you run the `scripts/kind-build-test.sh` script the first time,
    the step that builds the container image for the target ACK service
    controller can take up to 40 or more minutes. This is because the container image
    contains a lot of dependencies. Once you successfully build the target
    image this base image layer is cached locally, and the build takes a much 
    shorter amount of time. We are aware of this (and the storage footprint,
    ca. 3 GB) and aim to reduce both in the fullness of time.

In summary, in order to test ACK you will need to have the following tools
installed and configured:

1. [Docker](https://docs.docker.com/get-docker/)
1. [kind](https://kind.sigs.k8s.io/docs/user/quick-start/)
1. [kubernetes-sigs/controller-tools](https://github.com/kubernetes-sigs/controller-tools)
1. [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv1.html) version 1
1. [jq](https://github.com/stedolan/jq/wiki/Installation)
1. `make`

To build and test an ACK controller with `kind`, execute the commands as 
described in the following from the root directory of your
[checked-out source repository](../setup/).

!!! tip "Recommended RAM"
    Given that our test setup creates the container images and then launches
    a test cluster, we recommend that you have at least 4GB of RAM available
    for the tests.

With the prerequisites out of the way, let's move on to the first step:
building the code generator.

## Build code generator

To build the latest `ack-generate` binary, execute the following command:

```
make build-ack-generate
```

!!! note "One-off build"
    You only have to do this once, overall. In other words: unless we change
    something upstream in terms of the code generation process, this is 
    a one-off operation. Internally, the Makefile executes an `go build` here.

Don't worry if you forget this step, the script in the next step will complain 
with a message along the line of `ERROR: Unable to find an ack-generate binary`
and will give you another opportunity to rectify the situation.

## Build an ACK service controller

Now that we have the basic code generation step done we will create the
respective ACK service controller and its supporting artifacts.

So first you have to select a service that you want to build and test.
You do that by setting the `SERVICE` environment variable. Let's say we want 
to test the S3 service (creating an S3 bucket), so we would execute the
following:

```
export SERVICE=s3
```

Now we are in a position to generate the ACK service controller for the S3 API.

The following outputs the generated code to the `services/$SERVICE` directory:

```
make build-controller SERVICE=$SERVICE
```

!!! bug "Handle `controller-gen: command not found`"
    If you run into the `controller-gen: command not found` message when
    executing `make build-controller` then you want to check if the
    `controller-gen` binary is available in `$GOPATH/bin`, see also
    [`#234`](https://github.com/aws/aws-controllers-k8s/issues/234).

In addition to the ACK service controller code, above generates the
custom resource definition (CRD) manifests as well as the necessary RBAC
settings using the [`build-controller.sh`](https://github.com/aws/aws-controllers-k8s/blob/main/scripts/build-controller.sh)
script.

!!! note "Beyond dev preview"
    In future, this step will also generate the end-user install artifacts, that
    is the Helm chart etc. that can be used to install those CRD manifests, 
    and a deployment manifest that runs the ACK service controller in a pod.

Now that we have the generation part completed, we want to see if the
generated artifacts indeed are able to create an S3 bucket for us.

You don't have to do the next steps, if all you want to test is if the
generation works, however, for an end-to-end test, the next, final step is
a necessary one.

## Run tests

Time to run the end-to-end test.

### IAM setup

In order for the ACK service controller to manage the S3 bucket, it needs an
identity. In other words, it needs an IAM role that represents the ACK service
controller towards the S3 service.

First, define the name of the IAM role that will have the permission to manage
S3 buckets on your behalf:

```
export ACK_TEST_IAM_ROLE=Admin-k8s
```

Now we need to verify the IAM principal (likely an IAM user) that is going to
assume the IAM role `ACK_TEST_IAM_ROLE`. So to get its ARN, execute:

```
export ACK_TEST_PRINCIPAL_ARN=$(aws sts get-caller-identity --query 'Arn' --output text)
```

You can verify if that worked using `echo $ACK_TEST_PRINCIPAL_ARN` and that should
print something along the lines of `arn:aws:iam::1234567890121:user/ausername`.

Next up, create the IAM role, adding the necessary trust relationship to the role,
using the following commands:

```
cat > trust-policy.json << EOF
{
	"Version": "2012-10-17",
	"Statement": {
		"Effect": "Allow",
		"Principal": {
			"AWS": "$ACK_TEST_PRINCIPAL_ARN"
		},
		"Action": "sts:AssumeRole"
	}
}
EOF
```

Using above trust policy, we can now create the IAM role:

```
aws iam create-role \
    --role-name $ACK_TEST_IAM_ROLE \
    --assume-role-policy-document file://trust-policy.json
```

Now we're in the position to give the IAM role `ACK_TEST_IAM_ROLE` the permission
to handle S3 buckets for us, using:

```
aws iam attach-role-policy \
    --role-name $ACK_TEST_IAM_ROLE \
    --policy-arn "arn:aws:iam::aws:policy/AmazonS3FullAccess"
```

!!! tip "Access delegation in IAM"
    If you're not that familiar with IAM access delegation, we recommend you
    to peruse the [IAM documentation](https://docs.aws.amazon.com/IAM/latest/UserGuide/tutorial_cross-account-with-roles.html)

Next, in order for our test to generate [temporary credentials](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html)
we need to tell it to use the IAM role we created in the previous step.
To generate the IAM role ARN, do:

```
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query 'Account' --output text) && \
export AWS_ROLE_ARN=arn:aws:iam::${AWS_ACCOUNT_ID}:role/${ACK_TEST_IAM_ROLE}
```

!!! info 
     The tests uses the `generate_temp_creds` function from the
     `scripts/lib/aws.sh` script, executing effectively 
     ` aws sts assume-role --role-session-arn $AWS_ROLE_ARN --role-session-name $TEMP_ROLE `
     which fetches temporarily `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`,
     and an `AWS_SESSION_TOKEN` used in turn to authentication the ACK
     controller. The duration of the session token is 900 seconds (15 minutes).

Phew that was a lot to set up, but good news: you're almost there.

### Run end-to-end test

Before you proceed, make sure that you've done the IAM setup in the previous
step.

!!! warning "IAM troubles?!"
    If you try the following command and you see an error message containing
    something along the line of `AWS_ROLE_ARN is not defined.` then you know
    that somewhere in the IAM setup you either left out a step or one of the
    commands failed.

Now we're finally in the position to execute the end-to-end test:

```
make kind-test SERVICE=$SERVICE
```

This provisions a Kubernetes cluster using `kind`, builds a container image with
the ACK service controller, and loads the container image into the `kind` cluster.

It then installs the ACK service controller and related Kubernetes manifests into
the `kind` cluster using `kustomize build | kubectl apply -f -`.

Then, the above script runs a series of test scripts that call `kubectl`
and the `aws` CLI tools to verify that custom resources of the type managed by
the respective ACK service controller is created, updated and deleted
appropriately (still TODO).

Finally, it will run tests that create resources for the respective service
and verify if the resource has successfully created. In our example case it
should create an S3 bucket and then destroy it again, yielding something like
the following (edited down to the relevant parts):

```
...
./scripts/kind-build-test.sh -s s3
Using Kubernetes kindest/node:v1.16.9@sha256:7175872357bc85847ec4b1aba46ed1d12fa054c83ac7a8a11f5c268957fd5765
Creating k8s cluster using "kind" ...
No kind clusters found.
Created k8s cluster using "kind"
Building s3 docker image
Building 's3' controller docker image with tag: ack-s3-controller:ec452ed
sha256:c9cbcc028f2b7351d0507f8542ab88c80f9fb5a3b8b800feee8e362882833eef
Loading the images into the cluster
Image: "ack-s3-controller:ec452ed" with ID "sha256:c9cbcc028f2b7351d0507f8542ab88c80f9fb5a3b8b800feee8e362882833eef" not yet present on node "test-ccc3c7f1-worker", loading...
Image: "ack-s3-controller:ec452ed" with ID "sha256:c9cbcc028f2b7351d0507f8542ab88c80f9fb5a3b8b800feee8e362882833eef" not yet present on node "test-ccc3c7f1-control-plane", loading...
Loading CRD manifests for s3 into the cluster
customresourcedefinition.apiextensions.k8s.io/buckets.s3.services.k8s.aws created
Loading RBAC manifests for s3 into the cluster
clusterrole.rbac.authorization.k8s.io/ack-controller-role created
clusterrolebinding.rbac.authorization.k8s.io/ack-controller-rolebinding created
Loading service controller Deployment for s3 into the cluster
2020/08/18 09:51:46 Fixed the missing field by adding apiVersion: kustomize.config.k8s.io/v1beta1
Fixed the missing field by adding kind: Kustomization
namespace/ack-system created
deployment.apps/ack-s3-controller created
Running aws sts assume-role --role-arn arn:aws:iam::1234567890121:role/Admin-k8s, --role-session-name tmp-role-1b779de5  --duration-seconds 900,
Temporary credentials generated
deployment.apps/ack-s3-controller env updated
Added AWS Credentials to env vars map
======================================================================================================
To poke around your test manually:
export KUBECONFIG=/Users/hausenbl/ACK/upstream/aws-controllers-k8s/scripts/../build/tmp-test-ccc3c7f1/kubeconfig
kubectl get pods -A
======================================================================================================
bucket.s3.services.k8s.aws/ack-test-smoke-s3 created
{
  "Name": "ack-test-smoke-s3",
  "CreationDate": "2020-08-18T08:52:04+00:00"
}
bucket.s3.services.k8s.aws "ack-test-smoke-s3" deleted
smoke took 27 second(s)
🥑 Deleting k8s cluster using "kind"
Deleting cluster "test-ccc3c7f1" ...
```

As you can see, in above case the end-to-end test (creating cluster, deploying
ACK, applying custom resources, and tear-down) took less than 30 seconds. This
is for the warmed caches case.

#### Repeat for other services

We have end-to-end tests for all services listed in the `DEVELOPER-PREVIEW`,
`BETA` and `GA` release statuses in our [service listing](../services)
document. Simply replace your `SERVICE` environment variable with the name of a
supported service and re-run the IAM and test steps outlined above.

### Background

We use [mockery](https://github.com/vektra/mockery) for unit testing.
You can install it by following the guideline on mockery's GitHub or simply
by running our handy scirpt at `./scripts/install_mockery.sh` for general
Linux environments.


 We track testing in the umbrella [issue 6](https://github.com/aws/aws-controllers-k8s/issues/6).
 on GitHub. Use this issue as a starting point and if you create a new
 testing-related issue, mention it from there.

## Clean up

To clean up a `kind` cluster, including the container images and configuration 
files created by the script specifically for said test cluster, execute:

```
kind delete cluster --name $CLUSTER_NAME
```

If you want to delete all `kind` cluster running on your machine, use: 
```
make delete-all-kind-clusters
```

With this the testing is completed. Thanks for your time and we appreciate your
feedback.
