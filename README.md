# XFNLIB

`xfnlib` is a general library for working with crossplane composition
functions.

It's purpose is to abstract functionality that may or will become boilerplate in
many, if not all crossplane composition functions within the GiantSwarm
ecosystem

This library is being composed with the security of your clusters in mind.
In practice what this means is that whilst some functions may require additional
permissions to be granted for them to execute, the absolute bare minimal set is
documented (usually enough permissions to be able to `get`) and it is left to
you to decide which permissions are required or not.

> **Warning**
>
> This does not mitigate any changes that may be introduced by `upbound` through
> `crossplane-rbac-manager` or any upstream changes implemented by [#3718]

## Functionality

### Composite

The following functions are provided for working with composite resources

- `New` Should be called at the top of the `RunFunction`
- `ToResponse` Sets the desired composite and composed resources into the
  response and returns it back to your function.
- `AddDesired` Adds an object to the desired resources
- `ToUnstructured` Convert an object into an unstructured object
- `ToUnstructuredKubernetesObject` Wrap an object in a `crossplane-contrib/provider-kubernetes:Object type`
- `To` Convert objects from one type to another by passing it through
  `json.Marshal`

### Authentication

#### AWS

The following methods are available for authentication to AWS

- `GetAssumeRoleArn` Loads the AWS ProviderConfig and reads the role chain,
  returning the first element in the chain
- `Config` Sets up the AWS config for AssumeRole authentication

The AWS provider requires the service account the pod is running with to be
granted permissions to access the `ProviderConfig`. It also requires the
service account to be annotated to use `AssumeRole`.

At the very least this must look like the following:

<details>

<summary>Service account</summary>

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: SERVICE_ACCOUNT_NAME
  namespace: crossplane
  annotations:
    eks.amazonaws.com/role-arn: ASSUME_ROLE_ARN
```

</details>

<details>

<summary>Cluster role</summary>

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: aws-provider-config-access
rules:
  - apiGroups:
      - aws.upbound.io
    resources:
      - providerconfigs
    verbs:
      - get
```

</details>

<details>

<summary>Cluster role binding</summary>

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: aws-provider-config-access
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: aws-provider-config-access
subjects:
  - kind: ServiceAccount
    name: SERVICE_ACCOUNT_NAME
    namespace: crossplane
```

</details>

#### Kubernetes

- `Client` Get a kubernetes client using whatever authentication method is
  available. If inside the cluster, this will use the credentials linked to the
  service account the pod is running with. If outside the cluster, this wills
  use the current `kubeconfig` context.

## Known issues

There are no current known issues. If you think you've found one? Please raise a
[Bug report]

[#3718]: https://github.com/crossplane/crossplane/issues/3718
[Bug report]: https://github.com/giantswarm/xfnlib/issues/new?assignees=&labels=bug&projects=&template=bug_report.md