# wasmCloud-contrib

Community contributions of providers, components, and demos

## Components

- [moonbit/http-hello-world](./components/moonbit/http-hello-world/) is an example component that implements the `wasi-http/incoming-handler@0.2.0` interface and is built with [Moonbit](https://www.moonbitlang.com/)

## Secrets

There are currently two implementations of [wasmCloud secrets backends](https://wasmcloud.com/docs/deployment/security/secrets#implementing-a-secrets-backend) available in this repository.

- [secrets-kubernetes](./secrets/secrets-kubernetes/) for using secrets stored in Kubernetes in wasmCloud applications
- [secrets-vault](./secrets/secrets-vault/) for using secrets stored in Vault in wasmCloud applications
