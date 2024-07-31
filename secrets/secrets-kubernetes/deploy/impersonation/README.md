# RBAC example

- `default.yaml`: Allows the secrets backend to see all secrets in the `default` namespace, without impersonation.
- `cluster-wide.yaml`: Creates an impersonation target `wasmcloud-secrets-privileged`, which can read secrets in any namespace.

wadm snippets

```
spec:
  policies:
    - name: rust-hello-world-secrets-default
      type: policy.secret.wasmcloud.dev/v1alpha1
      properties:
        backend: kube
    - name: rust-hello-world-secrets-impersonation
      type: policy.secret.wasmcloud.dev/v1alpha1
      properties:
        backend: kube
        impersonate: wasmcloud-secrets-privileged
        namespace: kube-system

  components:
    - name: http-component
      type: component
      properties:
        image: ....
        secrets:
          # secret in 'kube-system' namespace
          - name: example-impersonated
            properties:
              policy: rust-hello-world-secrets-impersonation
              key: k3s-serving
              field: tls.crt
          # secret in 'default' namespace
          - name: example
            properties:
              policy: rust-hello-world-secrets-default
              key: cluster-secrets
              field: api-password
```
