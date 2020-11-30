apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ack-controller-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ack-controller
subjects:
- kind: ServiceAccount
  name: default
  namespace: ack-system
