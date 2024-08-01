use std::net::TcpListener;
use std::{collections::HashMap, net::SocketAddrV4};

use anyhow::Result;
use nkeys::{KeyPair, XKey};
use secrets_vault::{SubjectMapper, VaultConfig, VaultSecretsBackend};
use serde::Serialize;
use serde_json::json;
use testcontainers::{
    core::{Host as TestHost, WaitFor},
    runners::AsyncRunner,
    ContainerAsync, GenericImage, ImageExt,
};
use vaultrs::{
    api::auth::oidc::requests::{SetConfigurationRequest, SetRoleRequest},
    client::{VaultClient, VaultClientSettingsBuilder},
};
use wascap::jwt::{Claims, ClaimsBuilder, Component, Host};
use wasmcloud_secrets_types::{Application, Context as SecretsContext, SecretRequest};

const AUTH_METHOD_MOUNT: &str = "secrets-jwt";
const DEFAULT_AUDIENCE: &str = "Vault";
const SECRETS_BACKEND_PREFIX: &str = "wasmcloud.secrets";
const SECRETS_SERVICE_NAME: &str = "vault-test";
const SECRETS_ROLE_NAME: &str = "vault-test-role";
const SECRETS_ENGINE_MOUNT: &str = "secret";
const SECRETS_SECRET_NAME: &str = "test-secret";

const NATS_SERVER_PORT: u16 = 4222;
const VAULT_SERVER_PORT: u16 = 8200;
const VAULT_ROOT_TOKEN_ID: &str = "vault-root-token-id";

#[tokio::test]
async fn test_server_xkey() -> Result<()> {
    let xkey = nkeys::XKey::new();

    let nats_server = start_nats().await?;
    let nats_address = address_for_scheme_on_port(&nats_server, "nats", NATS_SERVER_PORT).await?;
    let nats_client = async_nats::connect(nats_address)
        .await
        .expect("connect to nats");

    let vault_server = start_vault(VAULT_ROOT_TOKEN_ID).await?;
    let vault_address =
        address_for_scheme_on_port(&vault_server, "http", VAULT_SERVER_PORT).await?;

    let jwks_port = find_open_port().await?;
    let jwks_address = format!("0.0.0.0:{jwks_port}").parse::<SocketAddrV4>()?;
    tokio::spawn({
        let vault_config = VaultConfig {
            address: vault_address,
            auth_mount: AUTH_METHOD_MOUNT.to_string(),
            jwt_audience: DEFAULT_AUDIENCE.to_string(),
            default_secret_engine: SECRETS_ENGINE_MOUNT.to_string(),
            default_namespace: None,
        };
        let subject_mapper = SubjectMapper::new(SECRETS_BACKEND_PREFIX, SECRETS_SERVICE_NAME)?;
        let secrets_nkey = nkeys::KeyPair::new_account();
        let secrets_xkey = nkeys::XKey::from_seed(&xkey.seed().unwrap()).unwrap();
        let secrets_nats_client = nats_client.clone();
        async move {
            VaultSecretsBackend::new(
                secrets_nats_client,
                secrets_nkey,
                secrets_xkey,
                jwks_address,
                subject_mapper,
                vault_config,
            )
            .serve()
            .await
        }
    });
    // Give the server a second to start before we query
    tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;

    let server_xkey_subject =
        format!("{SECRETS_BACKEND_PREFIX}.v1alpha1.{SECRETS_SERVICE_NAME}.server_xkey");
    let resp = nats_client
        .request(server_xkey_subject, "".into())
        .await
        .expect("request server_xkey via nats");

    let actual =
        std::str::from_utf8(&resp.payload).expect("convert server_xkey response payload to str");
    let expected = xkey.public_key();

    assert_eq!(actual, &expected);
    Ok(())
}

#[tokio::test]
async fn test_get() -> Result<()> {
    let nkey = nkeys::KeyPair::new_account();
    let xkey = nkeys::XKey::new();

    let nats_server = start_nats().await?;
    let nats_address = address_for_scheme_on_port(&nats_server, "nats", NATS_SERVER_PORT).await?;
    let nats_client = async_nats::connect(nats_address)
        .await
        .expect("connection to nats");

    let vault_server = start_vault(VAULT_ROOT_TOKEN_ID).await?;
    let vault_address =
        address_for_scheme_on_port(&vault_server, "http", VAULT_SERVER_PORT).await?;
    let vault_client = VaultClient::new(
        VaultClientSettingsBuilder::default()
            .address(&vault_address)
            .token(VAULT_ROOT_TOKEN_ID)
            .build()
            .expect("should build VaultClientSettings"),
    )
    .expect("should initialize a VaultClient");

    let jwks_port = find_open_port().await?;
    tokio::spawn({
        let vault_config = VaultConfig {
            address: vault_address,
            auth_mount: AUTH_METHOD_MOUNT.to_string(),
            jwt_audience: DEFAULT_AUDIENCE.to_string(),
            default_secret_engine: SECRETS_ENGINE_MOUNT.to_owned(),
            default_namespace: None,
        };
        let jwks_address = format!("0.0.0.0:{jwks_port}").parse::<SocketAddrV4>()?;
        let subject_mapper = SubjectMapper::new(SECRETS_BACKEND_PREFIX, SECRETS_SERVICE_NAME)?;
        let vault_xkey = nkeys::XKey::from_seed(&xkey.seed().unwrap()).unwrap();
        let vault_nats_client = nats_client.clone();
        async move {
            VaultSecretsBackend::new(
                vault_nats_client,
                nkey,
                vault_xkey,
                jwks_address,
                subject_mapper,
                vault_config,
            )
            .serve()
            .await
        }
    });
    // Give the server time to start before we query
    tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;

    configure_vault_jwt_auth(&vault_client, resolve_jwks_url(jwks_port)).await?;

    let secret_key = "secret-key";
    let stored_secret = HashMap::from([(secret_key, "this-is-a-secret")]);

    store_secret_in_engine_at_path(
        &vault_client,
        stored_secret.clone(),
        SECRETS_ENGINE_MOUNT,
        SECRETS_SECRET_NAME,
    )
    .await?;

    let wsc = wasmcloud_secrets_client::Client::new("vault-test", "wasmcloud.secrets", nats_client)
        .await
        .expect("should be able to instantiate wasmcloud-secrets-client");

    // TODO remove this once wascap uses the latest version of nkeys
    let claims_signer = wascap::prelude::KeyPair::new_account();
    let component_key = KeyPair::new_module();
    let host_key = KeyPair::new_server();
    let entity_claims: Claims<Component> = ClaimsBuilder::new()
        .issuer(claims_signer.public_key().as_str())
        .subject(component_key.public_key().as_str())
        .build();
    let host_claims: Claims<Host> = ClaimsBuilder::new()
        .issuer(claims_signer.public_key().as_str())
        .subject(host_key.public_key().as_str())
        .with_metadata(Host::new("test".to_string(), HashMap::new()))
        .build();

    let request_xkey = XKey::new();
    let secret_request = SecretRequest {
        key: SECRETS_SECRET_NAME.to_string(),
        field: Some(secret_key.to_owned()),
        version: None,
        context: SecretsContext {
            entity_jwt: entity_claims.encode(&claims_signer).unwrap(),
            host_jwt: host_claims.encode(&claims_signer).unwrap(),
            application: Application {
                name: Some("test-app".to_string()),
                policy: json!({
                    "type": "properties.secret.wasmcloud.dev/v1alpha1",
                    "properties": {
                        "roleName": SECRETS_ROLE_NAME,
                    }
                })
                .to_string(),
            },
        },
    };
    let secret = wsc
        .get(secret_request, request_xkey)
        .await
        .expect("should have gotten a secret");

    let actual = secret.string_secret.unwrap_or_default();
    let expected = stored_secret.get(secret_key).unwrap();

    assert_eq!(&actual, expected);

    Ok(())
}

async fn find_open_port() -> Result<u16> {
    let listener = TcpListener::bind("0.0.0.0:0")?;
    let socket_addr = listener.local_addr()?;
    Ok(socket_addr.port())
}

async fn start_nats() -> Result<ContainerAsync<GenericImage>> {
    Ok(GenericImage::new("nats", "2.10.16-linux")
        .with_exposed_port(NATS_SERVER_PORT.into())
        .with_wait_for(WaitFor::message_on_stderr("Server is ready"))
        .start()
        .await
        .expect("nats to start"))
}

async fn start_vault(root_token: &str) -> Result<ContainerAsync<GenericImage>> {
    let image = GenericImage::new("hashicorp/vault", "1.16.3")
        .with_exposed_port(VAULT_SERVER_PORT.into())
        .with_wait_for(WaitFor::message_on_stdout("==> Vault server started!"))
        .with_env_var("VAULT_DEV_ROOT_TOKEN_ID", root_token);
    Ok(image
        .with_host("host.docker.internal", TestHost::HostGateway)
        .start()
        .await
        .expect("vault to start"))
}

async fn address_for_scheme_on_port(
    service: &ContainerAsync<GenericImage>,
    scheme: &str,
    port: u16,
) -> Result<String> {
    Ok(format!(
        "{}://{}:{}",
        scheme,
        service.get_host().await?,
        service.get_host_port_ipv4(port).await?
    ))
}

async fn configure_vault_jwt_auth(vault_client: &VaultClient, jwks_url: String) -> Result<()> {
    // vault auth enable jwt
    vaultrs::sys::auth::enable(vault_client, AUTH_METHOD_MOUNT, "jwt", None)
        .await
        .unwrap_or_else(|_| {
            panic!(
                "should have enabled the 'jwt' auth method at '{}'",
                AUTH_METHOD_MOUNT
            )
        });

    // vault write auth/<engine-path>/config jwks_url="http://localhost:3000/.well-known/keys"
    let mut config_builder = SetConfigurationRequest::builder();
    config_builder.jwks_url(jwks_url.clone());

    vaultrs::auth::oidc::config::set(vault_client, AUTH_METHOD_MOUNT, Some(&mut config_builder))
        .await
        .unwrap_or_else(|_| panic!("should have configured the 'jwt' auth method at '{}' with the default role '{}' and jwks_url '{}'", AUTH_METHOD_MOUNT, SECRETS_ROLE_NAME, jwks_url));

    // cat role-config.json | vault write auth/jwt/role/test-role -
    let user_claim = "sub";
    let allowed_redirect_uris = vec![];
    let mut role_builder = SetRoleRequest::builder();
    role_builder
        .role_type("jwt")
        .bound_audiences(vec!["Vault".to_string()])
        .bound_claims(HashMap::from([(
            "application".to_string(),
            "test-app".to_string(),
        )]))
        .token_policies(vec![SECRETS_ROLE_NAME.to_string()]);
    vaultrs::auth::oidc::role::set(
        vault_client,
        AUTH_METHOD_MOUNT,
        SECRETS_ROLE_NAME,
        user_claim,
        allowed_redirect_uris,
        Some(&mut role_builder),
    )
    .await
    .unwrap_or_else(|_| {
        panic!(
            "should have configured the default role '{}' for 'jwt' auth method",
            SECRETS_ROLE_NAME
        )
    });

    // vault policy set ...
    let policy = r#"
    path "secret/*" {
        capabilities = ["create", "read", "update", "delete", "list"]
    }"#;
    vaultrs::sys::policy::set(vault_client, SECRETS_ROLE_NAME, policy)
        .await
        .unwrap_or_else(|_| {
            panic!(
                "should have set up policy for the '{}' role",
                SECRETS_ROLE_NAME
            )
        });
    Ok(())
}

async fn store_secret_in_engine_at_path(
    vault_client: &VaultClient,
    value: impl Serialize,
    mount: &str,
    path: &str,
) -> Result<()> {
    vaultrs::kv2::set(vault_client, mount, path, &value).await?;
    Ok(())
}

// Resolves the platform-specific endpoint where the docker containers can
// reach the host in order for the Vault Server running inside the docker
// container can connect to the JWKS endpoint exposed by the Vault Secret Backend.
fn resolve_jwks_url(jwks_port: u16) -> String {
    // TODO: Add the option to provide a configuration option via environment
    // variable, or fall back to one of the OS-specific defaults below.
    #[cfg(target_os = "linux")]
    {
        // Default bridge network IP set up by Docker:
        // https://docs.docker.com/network/network-tutorial-standalone/#use-the-default-bridge-network
        format!("http://172.17.0.1:{}/.well-known/keys", jwks_port)
    }
    #[cfg(target_os = "macos")]
    {
        // Magic hostname set up by Docker for Mac Desktop:
        // https://docs.docker.com/desktop/networking/#i-want-to-connect-from-a-container-to-a-service-on-the-host
        format!("http://host.docker.internal:{}/.well-known/keys", jwks_port)
    }
}
