# Kubernetes Secrets Backend Implementation for [wasmCloud Secrets](https://github.com/wasmCloud/wasmCloud/issues/2190)

## Basic Usage

Get secret `app-secrets` in the `default` namespace, and expose secret key `some-password` as `some_password` to component.

```yaml
spec:
  policies:
    - name: rust-hello-world-secrets-default
      type: policy.secret.wasmcloud.dev/v1alpha1
      properties:
        backend: kube
  components:
    - name: http-component
      type: component
      properties:
        image: .......
        secrets:
          - name: some_password
            properties:
              policy: rust-hello-world-secrets-default
              key: app-secrets
              field: some-password
```

## Advanced Usage ( Impersonation )

Get secret `cluster-secrets` in the `kube-system` namespace, and expose secret key `tls.crt` as `cluster_certificate` to component.
The backend will impersonate the `wasmcloud-secrets-privileged` ClusterRole, defined in `impersonate`.

```yaml
spec:
  policies:
    - name: rust-hello-world-secrets-impersonation
      type: policy.secret.wasmcloud.dev/v1alpha1
      properties:
        backend: kube
        # Cluster Role to Impersonate
        impersonate: wasmcloud-secrets-privileged
        # Namespace to retrieve secrets from
        namespace: kube-system
  components:
    - name: http-component
      type: component
      properties:
        image: ...
        secrets:
          - name: cluster_certificate
            properties:
              policy: rust-hello-world-secrets-impersonation
              key: cluster-secrets
              field: tls.crt
```

## Machinery

- wasmCloud Secrets Protocol ( `server_xkey` and `get` operations )
- wasCap jwt validation using Ed25519
- wasCap Host & Entity Capabilities unwrapping

The `pkg/secrets` can be used to implement other Secrets Backends via `secrets.NewServer()` and its `secrets.Handler` companion.

```go
type secretProvider struct{}
func (s *secretProvider) Get(ctx context.Context, r *secrets.Request) (*secrets.SecretValue, error) {
return &secrets.SecretValue{
    StringSecret: "p@$$w0rd",
    Version:      "latest",
}, nil
}

...
provider := &secretProvider{}
// create a secrets server with ephemeral curve key
// can also pass stable key with `secrets.WithKeyPair(nkeys.KeyPair)`
secretsServer, _ := secrets.NewServer("provider-name", natsConnection, provider, secrets.WithEphemeralKey())

// Start secrets server
secretsServer.Run()

// Shutdown secrets server
// Receives a boolean to drain in-flight messages or bail quickly
secretsServer.Shutdown(true)
```

## Installation

### Automated

Refer to the [wasmcloud operator helm chart](https://github.com/wasmCloud/wasmcloud-operator).

### Manual

See [deploy/dev](deploy/dev) for a deployment example. You will need:

- NATS URL
- A NATS Curve Key to encrypt data between secrets backend & wasmcloud hosts

Generate a Curve key, take note of the generated 'Seed':

```bash
wash keys gen Curve
```

These values should be passed to the container as:

```yaml
args:
  - "--backend-seed=$(BACKEND_SEED)"
  - "--nats-url=$(NATS_URL)"
```

Next step is to give permissions to the service account running the secrets backend. Assuming service account `default` and namespace `wasmcloud-secrets`, give permission to read secrets in the `default` namespace:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: wasmcloud-secrets-reader-default
  namespace: default
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "watch", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: wasmcloud-secrets-reader-default
  namespace: default
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: wasmcloud-secrets-reader-default
subjects:
  - apiGroup: rbac.authorization.k8s.io
    kind: User
    name: "system:serviceaccount:wasmcloud-secrets:default"
```

## Local Development

Create `deploy/dev/kubernetes-backend.env` using the provided sample (kubernetes-backend.env.sample).

Build Container Image with `make build`

Deploy manifests with `make dev-init`

Iterate deploys with `make dev-deploy`. This will build & restart containers.

See pod logs with `make dev-logs`
