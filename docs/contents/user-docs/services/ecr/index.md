# Elastic Container Registry

The following walks you through creating an AWS Elastic Container Registry using the ECR controller.

To use this example you will need to have the ECR controller deployed with the correct IAM permissions to create an example ECR repo.
During the development preview can follow the [development testing documentation](../../../dev-docs/testing.md) to build the ECR controller and deploy it to a Kubernetes cluster.

## Example Kubernetes manifests

There are example Custom Resource Definitions (CRD) to create an ECR repo available with the service controller code.
To create a simple repository you can use the following manifest

```yaml
---
apiVersion: "ecr.services.k8s.aws/v1alpha1"
kind: Repository
metadata:
  name: "test-repository-from-ack"
spec:
  repositoryName: "test-repository-from-ack"
  tags:
  - key: "is-encrypted"
    value: "false"
```

To deploy this to your Kubernetes cluster you can use the sample file.

```sh
kubectl apply -f https://github.com/aws/aws-controllers-k8s/blob/examples/services/ecr/config/samples/example-repo.yaml
```

More advanced example use cases are available in the [ECR samples directory](https://github.com/aws/aws-controllers-k8s/tree/examples/services/ecr/config/samples).