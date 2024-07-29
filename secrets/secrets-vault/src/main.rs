use std::net::SocketAddrV4;

use clap::{command, Parser};
use nkeys::{KeyPair, XKey};
use secrets_vault::{SubjectMapper, VaultConfig, VaultSecretsBackend};

#[derive(Parser, Debug)]
#[command(version, about, long_about = None)]
struct Args {
    #[command(flatten)]
    pub nats_client_opts: NatsClientOpts,

    #[command(flatten)]
    pub secrets_server_opts: SecretsServerOpts,

    #[command(flatten)]
    pub vault_opts: VaultOpts,
}

#[derive(Parser, Debug)]
struct NatsClientOpts {
    /// NATS Server address to connect to listen for secrets requests.
    #[arg(long = "nats-address", env = "SV_NATS_ADDRESS")]
    pub address: String,

    /// JWT for authenticating the NATS connection
    #[arg(long = "nats-jwt", env = "SV_NATS_JWT", requires = "seed")]
    pub jwt: Option<String>,

    /// NATS Seed for signing the nonce for JWT authentication. Can be the same as server-nkey-seed.
    #[arg(long = "nats-seed", env = "SV_NATS_SEED", requires = "jwt")]
    pub seed: Option<String>,
}

impl NatsClientOpts {
    pub(crate) async fn into_nats_client(self) -> anyhow::Result<async_nats::Client> {
        let mut options = async_nats::ConnectOptions::new();
        if self.jwt.is_some() && self.seed.is_some() {
            let keypair = std::sync::Arc::new(KeyPair::from_seed(&self.seed.unwrap())?);
            options = options.jwt(self.jwt.unwrap(), move |nonce| {
                let kp = keypair.clone();
                async move { kp.sign(&nonce).map_err(async_nats::AuthError::new) }
            });
        }
        Ok(async_nats::connect_with_options(self.address, options).await?)
    }
}

#[derive(Parser, Debug)]
struct SecretsServerOpts {
    /// Address for serving the JWKS endpoint, for example: 127.0.0.1:8080
    #[arg(long = "jwks-address", env = "SV_JWKS_ADDRESS")]
    pub jwks_address: SocketAddrV4,

    /// Nkey to be used for representing the Server's identity to Vault. Used for JWKS and signing payloads.
    #[arg(long = "server-nkey-seed", env = "SV_SERVER_NKEY_SEED")]
    pub nkey_seed: String,

    /// Xkey seed to be used to encrypt communication from hosts to the backend, this will be used to serve the public key via `server_xkey` operation.
    #[arg(long = "server-xkey-seed", env = "SV_SERVER_XKEY_SEED")]
    pub xkey_seed: String,

    /// Secrets subject prefix to listen on. Defaults to `wasmcloud.secrets`.
    #[arg(
        long = "secrets-prefix",
        env = "SV_SECRETS_PREFIX",
        default_value = "wasmcloud.secrets"
    )]
    pub prefix: String,

    /// Service name to be used to identify the subject this backend should listen on for secrets requests.
    #[arg(
        long = "secrets-service-name",
        env = "SV_SERVICE_NAME",
        default_value = "vault"
    )]
    pub service_name: String,
}

#[derive(Parser, Debug)]
struct VaultOpts {
    /// Vault server address to connect to.
    #[arg(long = "vault-address", env = "SV_VAULT_ADDRESS")]
    pub address: String,

    /// Path where the JWT auth method is mounted
    #[arg(long = "vault-auth-method", env = "SV_VAULT_AUTH_METHOD")]
    pub auth_method_path: String,

    /// JWT (aud)ience value to be passed to Vault as part of authentication
    #[arg(
        long = "jwt-auth-audience",
        env = "SV_JWT_AUTH_AUDIENCE",
        default_value = "Vault"
    )]
    pub jwt_audience: String,

    /// Default secret engine path to use. This can be overridden on a per-request basis.
    #[arg(
        long = "vault-default-secret-engine",
        env = "SV_VAULT_DEFAULT_SECRET_ENGINE"
    )]
    pub default_secret_engine: String,

    /// Optional: Default Vault namespace to use, only use this with Vault Enterprise.
    #[arg(long = "vault-default-namespace", env = "SV_VAULT_DEFAULT_NAMESPACE")]
    pub default_namespace: Option<String>,
}

impl From<VaultOpts> for VaultConfig {
    fn from(opts: VaultOpts) -> Self {
        Self {
            address: opts.address,
            auth_mount: opts.auth_method_path.trim_matches('/').to_owned(),
            jwt_audience: opts.jwt_audience,
            default_secret_engine: opts.default_secret_engine.trim_matches('/').to_owned(),
            default_namespace: opts.default_namespace,
        }
    }
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let args = Args::parse();

    tracing_subscriber::fmt::init();

    let nkey = match KeyPair::from_seed(&args.secrets_server_opts.nkey_seed) {
        Ok(nk) => nk,
        Err(e) => anyhow::bail!("Could not parse provided NKey: {e}"),
    };
    let xkey = match XKey::from_seed(&args.secrets_server_opts.xkey_seed) {
        Ok(nk) => nk,
        Err(e) => anyhow::bail!("Could not parse provided XKey: {e}"),
    };

    let subject_mapper = SubjectMapper::new(
        &args.secrets_server_opts.prefix,
        &args.secrets_server_opts.service_name,
    )?;

    let nats_client = match args.nats_client_opts.into_nats_client().await {
        Ok(nc) => nc,
        Err(e) => anyhow::bail!("Could not connect to NATS with the provided configuration: {e}"),
    };

    let vault_config = VaultConfig::from(args.vault_opts);

    let backend = VaultSecretsBackend::new(
        nats_client,
        nkey,
        xkey,
        args.secrets_server_opts.jwks_address,
        subject_mapper,
        vault_config,
    );

    backend.serve().await?;
    Ok(())
}
