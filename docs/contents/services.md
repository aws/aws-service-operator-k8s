# Services

The following AWS service APIs have service controllers included in ACK or have
controllers currently being built.

For details, including a list of planned AWS service APIs, see the [Service
Controller Release Roadmap](https://github.com/aws/aws-controllers-k8s/projects/1):

!!! note "IMPORTANT"
    There is no single release of the ACK project. The ACK project contains a
    series of service controllers, one for each AWS service API. Each
    individual ACK service controller is released separately. Please see the
    documentation on [release criteria](releases.md) for information on how we
    release ACK service controllers.

| AWS Service | Current Status | Next Milestone
| ----------- | -------------- | --------------
| Amazon [API Gateway V2][apigwv2] | `DEVELOPER PREVIEW` | [`BETA`](https://github.com/aws/aws-controllers-k8s/milestone/15)
| Amazon [CloudFront Distribution][cfd] | `PLANNED` | [`DEVELOPER PREVIEW`](https://github.com/aws/aws-controllers-k8s/milestone/14)
| Amazon [DynamoDB][dynamodb] | `DEVELOPER PREVIEW` |
| Amazon [ECR][ecr] | `DEVELOPER PREVIEW` |
| Amazon [EFS][efs] | `PROPOSED` |
| Amazon [EKS][eks] | `PLANNED` | [`DEVELOPER PREVIEW`](https://github.com/aws/aws-controllers-k8s/milestone/7)
| Amazon [ElastiCache][elasticache] | `DEVELOPER PREVIEW` | [`BETA`](https://github.com/aws/aws-controllers-k8s/milestone/9)
| AWS [IAM][iam] | `PROPOSED` |
| AWS [Lambda][lambda] | `PLANNED` | [`DEVELOPER PREVIEW`](https://github.com/aws/aws-controllers-k8s/milestone/10)
| Amazon [MQ][mq] | `PLANNED` | [`DEVELOPER PREVIEW`](https://github.com/aws/aws-controllers-k8s/milestone/12)
| Amazon [MSK][msk] | `PLANNED` | [`DEVELOPER PREVIEW`](https://github.com/aws/aws-controllers-k8s/milestone/13)
| Amazon [RDS][rds] | `PLANNED` | [`DEVELOPER PREVIEW`](https://github.com/aws/aws-controllers-k8s/milestone/8)
| Amazon [Sagemaker][sagemaker] | `BUILD` | [`DEVELOPER PREVIEW`](https://github.com/aws/aws-controllers-k8s/milestone/11)
| Amazon [SNS][sns] | `DEVELOPER PREVIEW` |
| Amazon [SQS][sqs] | `DEVELOPER PREVIEW` |
| AWS [StepFunctions][sfn] | `DEVELOPER PREVIEW`
| Amazon [S3][s3] | `DEVELOPER PREVIEW` |

!!! note "Don't see a service listed?"
    If you don't see a particular AWS service listed, feel free to
    [propose it](https://github.com/aws/aws-controllers-k8s/issues/new?labels=Service+Controller&template=propose_new_controller.md&title=%5Bname%5D+service+controller)!

## Amazon API Gateway v2 [apigwv2]

* Proposed: https://github.com/aws/aws-controllers-k8s/issues/207
* Current release status: `DEVELOPER PREVIEW`
* Next milestone: [`BETA`](https://github.com/aws/aws-controllers-k8s/milestone/15)
* AWS service documentation: https://aws.amazon.com/api-gateway/
* ACK service controller: https://github.com/aws/aws-controllers-k8s/tree/main/services/apigatewayv2

## Amazon CloudFront Distribution [cfd]

* Proposed: https://github.com/aws/aws-controllers-k8s/issues/249
* Next milestone: [`DEVELOPER PREVIEW`](https://github.com/aws/aws-controllers-k8s/milestone/14)
* Current release status: `PLANNED`
* AWS service documentation: https://aws.amazon.com/cloudfront/

## Amazon DynamoDB [dynamodb]

* Proposed: https://github.com/aws/aws-controllers-k8s/issues/206
* Current release status: `DEVELOPER PREVIEW`
* Next milestone: [`BETA`](https://github.com/aws/aws-controllers-k8s/milestone/16)
* AWS service documentation: https://aws.amazon.com/dynamodb/
* ACK service controller: https://github.com/aws/aws-controllers-k8s/tree/main/services/dynamodb

## Amazon ECR [ecr]

* Proposed: https://github.com/aws/aws-controllers-k8s/issues/208
* Current release status: `DEVELOPER PREVIEW`
* Next milestone: [`BETA`](https://github.com/aws/aws-controllers-k8s/milestone/16)
* AWS service documentation: https://aws.amazon.com/ecr/
* ACK service controller: https://github.com/aws/aws-controllers-k8s/tree/main/services/ecr

## Amazon EFS [efs]

* Proposed: https://github.com/aws/aws-controllers-k8s/issues/328
* Current release status: `PROPOSED`
* AWS service documentation: https://aws.amazon.com/efs/

## Amazon EKS [eks]

* Proposed: https://github.com/aws/aws-controllers-k8s/issues/16
* Current release status: `PLANNED`
* AWS service documentation: https://aws.amazon.com/eks/

## Amazon Elasticache [elasticache]

* Proposed: https://github.com/aws/aws-controllers-k8s/issues/240
* Current release status: `DEVELOPER PREVIEW`
* Next milestone: [`BETA`](https://github.com/aws/aws-controllers-k8s/milestone/9)
* AWS service documentation: https://aws.amazon.com/elasticache/
* ACK service controller: https://github.com/aws/aws-controllers-k8s/tree/main/services/elasticache

## AWS IAM [iam]

* Proposed: https://github.com/aws/aws-controllers-k8s/issues/222
* Current release status: `PROPOSED`
* AWS service documentation: https://aws.amazon.com/iam/

## AWS Lambda [lambda]

* Proposed: https://github.com/aws/aws-controllers-k8s/issues/238
* Current release status: `PLANNED`
* Next milestone: [`DEVELOPER PREVIEW`](https://github.com/aws/aws-controllers-k8s/milestone/10)
* AWS service documentation: https://aws.amazon.com/lambda/

## Amazon MQ [mq]

* Proposed: https://github.com/aws/aws-controllers-k8s/issues/390
* Current release status: `PLANNED`
* Next milestone: [`DEVELOPER PREVIEW`](https://github.com/aws/aws-controllers-k8s/milestone/12)
* AWS service documentation: https://aws.amazon.com/amazon-mq/

## Amazon MSK [msk]

* Proposed: https://github.com/aws/aws-controllers-k8s/issues/348
* Current release status: `PLANNED`
* Next milestone: [`DEVELOPER PREVIEW`](https://github.com/aws/aws-controllers-k8s/milestone/13)
* AWS service documentation: https://aws.amazon.com/msk/

## Amazon RDS [rds]

* Proposed: https://github.com/aws/aws-controllers-k8s/issues/237
* Current release status: `PLANNED`
* Next milestone: [`DEVELOPER PREVIEW`](https://github.com/aws/aws-controllers-k8s/milestone/8)
* AWS service documentation: https://aws.amazon.com/rds/

## Amazon Sagemaker [sagemaker]

* Proposed: https://github.com/aws/aws-controllers-k8s/issues/385
* Current release status: `BUILD`
* Next milestone: [`DEVELOPER PREVIEW`](https://github.com/aws/aws-controllers-k8s/milestone/11)
* AWS service documentation: https://aws.amazon.com/sagemaker/

## Amazon SNS [sns]

* Proposed: https://github.com/aws/aws-controllers-k8s/issues/202
* Current release status: `DEVELOPER PREVIEW`
* Next milestone: [`BETA`](https://github.com/aws/aws-controllers-k8s/milestone/17)
* AWS service documentation: https://aws.amazon.com/sns/
* ACK service controller: https://github.com/aws/aws-controllers-k8s/tree/main/services/sns

## Amazon SQS [sqs]

* Proposed: https://github.com/aws/aws-controllers-k8s/issues/205
* Current release status: `DEVELOPER PREVIEW`
* AWS service documentation: https://aws.amazon.com/sqs/
* ACK service controller: https://github.com/aws/aws-controllers-k8s/tree/main/services/sqs

## AWS Step Functions [sfn]

* Proposed: https://github.com/aws/aws-controllers-k8s/issues/239
* Current release status: `DEVELOPER PREVIEW`
* AWS service documentation: https://aws.amazon.com/step-functions/
* ACK service controller: https://github.com/aws/aws-controllers-k8s/tree/main/services/sfn

## Amazon S3 [s3]

* Proposed: https://github.com/aws/aws-controllers-k8s/issues/204
* Current release status: `DEVELOPER PREVIEW`
* Next milestone: [`BETA`](https://github.com/aws/aws-controllers-k8s/milestone/16)
* AWS service documentation: https://aws.amazon.com/s3/
* ACK service controller: https://github.com/aws/aws-controllers-k8s/tree/main/services/s3
