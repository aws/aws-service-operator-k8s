# ECR Resources

The ECR controller managed Elastic Container Registry resources inside AWS.

Once you have the controller deployed and a custom resource definition defined in your cluster you can try deploying some test ECR repositories with the manifests in this folder.

### Simple ECR repo
```
kubectl apply -f https://raw.githubusercontent.com/aws/aws-controllers-k8s/main/examples/ecr/ecr-repo.yaml
```

### Encrypted ECR repo
```
kubectl apply -f https://raw.githubusercontent.com/aws/aws-controllers-k8s/main/examples/ecr/ecr-encrypted-repo.yaml
```

### Fully customized ECR repo
```
kubectl apply -f https://raw.githubusercontent.com/aws/aws-controllers-k8s/main/examples/ecr/ecr-full-repo.yaml
```