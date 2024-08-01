use anyhow::{anyhow, Result};
use async_nats::{HeaderMap, Message, Subject};
use ed25519_dalek::pkcs8::EncodePrivateKey;
use futures::StreamExt;
use jsonwebtoken::{get_current_timestamp, Algorithm, EncodingKey};
use nkeys::{KeyPair, XKey};
use serde::{Deserialize, Serialize};
use std::{net::SocketAddrV4, result::Result as StdResult, str::FromStr};
use tracing::info;
use vaultrs::{
    api::kv2::{requests::ReadSecretRequest, responses::ReadSecretResponse},
    client::{Client, VaultClient, VaultClientSettingsBuilder},
};
use wascap::{
    jwt::{CapabilityProvider, Component, Host},
    prelude::Claims,
};
use wasmcloud_secrets_types::{
    GetSecretError, Secret, SecretRequest, SecretResponse, RESPONSE_XKEY, WASMCLOUD_HOST_XKEY,
};

mod jwk;
mod jwks;
use jwks::VaultSecretsJwksServer;

const SECRETS_API_VERSION: &str = "v1alpha1";

#[derive(Debug, PartialEq)]
pub(crate) enum Operation {
    Get,
    ServerXkey,
}

impl FromStr for Operation {
    type Err = anyhow::Error;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s {
            "get" => Ok(Self::Get),
            "server_xkey" => Ok(Self::ServerXkey),
            subject => {
                anyhow::bail!("unsupported subject: {subject:?}")
            }
        }
    }
}

struct RequestClaims {
    host: Claims<Host>,
    component: Option<Claims<Component>>,
    provider: Option<Claims<CapabilityProvider>>,
}

impl TryFrom<&SecretRequest> for RequestClaims {
    type Error = anyhow::Error;

    fn try_from(request: &SecretRequest) -> StdResult<Self, Self::Error> {
        let host = Claims::<Host>::decode(&request.context.host_jwt)
            .map_err(|_| anyhow!("failed to decode host claims"))?;

        let component = Claims::<Component>::decode(&request.context.entity_jwt);
        let provider = Claims::<CapabilityProvider>::decode(&request.context.entity_jwt);

        if component.is_err() && provider.is_err() {
            return Err(anyhow!("failed to decode component and provider claims"));
        }

        Ok(Self {
            host,
            component: component.ok(),
            provider: provider.ok(),
        })
    }
}

impl RequestClaims {
    pub(crate) fn entity_id(&self) -> String {
        if let Some(component) = &self.component {
            component.id.clone()
        } else if let Some(provider) = &self.provider {
            provider.id.clone()
        } else {
            "Unknown".to_string()
        }
    }
}

#[derive(Serialize, Deserialize)]
struct VaultPolicy {
    #[serde(alias = "roleName")]
    role_name: String,
    #[serde(alias = "secretEnginePath")]
    secret_engine_path: Option<String>,
    namespace: Option<String>,
}

impl TryFrom<&SecretRequest> for VaultPolicy {
    type Error = anyhow::Error;

    fn try_from(request: &SecretRequest) -> StdResult<Self, Self::Error> {
        let policy = serde_json::from_str::<serde_json::Value>(&request.context.application.policy)
            .map_err(|e| anyhow!("failed to extract policy: {}", e.to_string()))?;
        let properties = policy
            .get("properties")
            .ok_or_else(|| anyhow!("failed to extract policy properties"))?;
        serde_json::from_str::<Self>(&properties.to_string()).map_err(|e| {
            anyhow!(
                "failed to deserialize vault policy from properties: {}",
                e.to_string()
            )
        })
    }
}

#[derive(Serialize, Deserialize)]
struct VaultAuthClaims {
    aud: String,
    iss: String,
    sub: String,
    exp: u64,
    application: String,
    host: Claims<Host>,
    #[serde(skip_serializing_if = "Option::is_none")]
    component: Option<Claims<Component>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    provider: Option<Claims<CapabilityProvider>>,
}

struct VaultSecretRef {
    path: String,
    field: Option<String>,
    version: Option<u64>,
}

impl TryFrom<&SecretRequest> for VaultSecretRef {
    type Error = anyhow::Error;

    fn try_from(request: &SecretRequest) -> StdResult<Self, Self::Error> {
        let version = request
            .version
            .to_owned()
            .map(|v| {
                v.parse::<u64>()
                    .map_err(|_| anyhow!("unable to convert requested version to integer"))
            })
            .transpose()?;

        Ok(Self {
            path: request.key.to_owned(),
            field: request.field.to_owned(),
            version,
        })
    }
}

pub struct SubjectMapper {
    pub prefix: String,
    pub service_name: String,
}

impl SubjectMapper {
    pub fn new(prefix: &str, service_name: &str) -> Result<Self> {
        Ok(Self {
            prefix: prefix.to_string(),
            service_name: service_name.to_string(),
        })
    }

    fn queue_group_name(&self) -> String {
        format!("{}.{}", self.prefix, self.service_name)
    }

    fn secrets_subject(&self) -> String {
        format!(
            "{}.{}.{}",
            self.prefix, SECRETS_API_VERSION, self.service_name
        )
    }

    fn secrets_wildcard_subject(&self) -> String {
        format!("{}.>", self.secrets_subject())
    }
}

pub struct VaultSecretsBackend {
    nats_client: async_nats::Client,
    nkey: nkeys::KeyPair,
    xkey: nkeys::XKey,
    jwks_address: SocketAddrV4,
    subject_mapper: SubjectMapper,
    vault_config: VaultConfig,
}

pub struct VaultConfig {
    pub address: String,
    pub auth_mount: String,
    pub jwt_audience: String,
    pub default_secret_engine: String,
    pub default_namespace: Option<String>,
}

impl VaultSecretsBackend {
    pub fn new(
        nats_client: async_nats::Client,
        nkey: KeyPair,
        xkey: XKey,
        jwks_address: SocketAddrV4,
        subject_mapper: SubjectMapper,
        vault_config: VaultConfig,
    ) -> Self {
        Self {
            nats_client,
            nkey,
            xkey,
            jwks_address,
            subject_mapper,
            vault_config,
        }
    }

    pub async fn serve(&self) -> anyhow::Result<()> {
        let pk = KeyPair::from_public_key(&self.nkey.public_key())?;
        tokio::spawn({
            let listen_address = self.jwks_address.to_owned();
            async move {
                VaultSecretsJwksServer::new(vec![pk], listen_address)
                    .unwrap()
                    .serve()
                    .await
                    .unwrap()
            }
        });
        self.start_nats_subscriber().await?;
        Ok(())
    }

    async fn start_nats_subscriber(&self) -> Result<()> {
        info!(
            "Subscribing to messages addressed to {} under queue group {}",
            self.subject_mapper.secrets_wildcard_subject(),
            self.subject_mapper.queue_group_name(),
        );

        let subject_prefix = self.subject_mapper.secrets_subject();
        let mut subscriber = self
            .nats_client
            .queue_subscribe(
                self.subject_mapper.secrets_wildcard_subject(),
                self.subject_mapper.queue_group_name(),
            )
            .await?;

        while let Some(message) = subscriber.next().await {
            // We check to see if there's a reply inbox, otherwise just ignore the message.
            let Some(reply_to) = message.reply.clone() else {
                continue;
            };

            match parse_op_from_subject(&message.subject, &subject_prefix) {
                Ok(Operation::Get) => {
                    if let Err(err) = self.handle_get_request(reply_to.clone(), message).await {
                        self.handle_get_request_error(reply_to, err).await;
                    }
                }
                Ok(Operation::ServerXkey) => {
                    self.handle_server_xkey_request(reply_to).await;
                }
                Err(err) => {
                    self.handle_unsupported_request(reply_to, err).await;
                }
            }
        }

        Ok(())
    }

    async fn handle_unsupported_request(&self, reply_to: Subject, error: GetSecretError) {
        // TODO: handle the potential publish error
        let _ = self
            .nats_client
            .publish(reply_to, SecretResponse::from(error).into())
            .await;
    }

    async fn handle_get_request_error(&self, reply_to: Subject, error: GetSecretError) {
        // TODO: handle the potential publish error
        let _ = self
            .nats_client
            .publish(reply_to, SecretResponse::from(error).into())
            .await;
    }

    async fn handle_get_request(
        &self,
        reply_to: Subject,
        message: Message,
    ) -> StdResult<(), GetSecretError> {
        if message.payload.is_empty() {
            return Err(GetSecretError::Other("missing payload".to_string()));
        }

        let host_xkey = Self::extract_host_xkey(&message)?;

        let secret_request = Self::extract_secret_request(&message, &self.xkey, &host_xkey)?;

        let request_claims = Self::validate_and_extract_claims(&secret_request)?;

        let policy = Self::validate_and_extract_policy(&secret_request)?;

        let auth_claims = VaultAuthClaims {
            iss: self.nkey.public_key(),
            aud: self.vault_config.jwt_audience.clone(),
            sub: request_claims.entity_id(),
            exp: get_current_timestamp() + 60,
            application: secret_request
                .context
                .application
                .name
                .clone()
                .unwrap_or_default(),
            host: request_claims.host,
            component: request_claims.component,
            provider: request_claims.provider,
        };

        let encoding_key = Self::convert_nkey_to_encoding_key(&self.nkey)?;

        let auth_jwt = Self::encode_claims_to_jwt(auth_claims, &encoding_key)?;

        let vault_client =
            Self::authenticate_with_vault(&self.vault_config, &policy, &auth_jwt).await?;

        let secret_ref = VaultSecretRef::try_from(&secret_request)
            .map_err(|e| GetSecretError::Other(e.to_string()))?;

        let response = Self::fetch_secret(
            &vault_client,
            &policy
                .secret_engine_path
                .unwrap_or_else(|| self.vault_config.default_secret_engine.to_owned()),
            &secret_ref,
        )
        .await?;

        let secret_version = response.metadata.version.to_string();
        let secret = if let Some(field) = secret_ref.field {
            response
                .data
                .get(field)
                .map(|v| v.as_str().unwrap())
                .map(ToString::to_string)
        } else {
            Some(response.data.to_string())
        };

        let secret_response = SecretResponse {
            secret: Some(Secret {
                version: secret_version,
                string_secret: secret,
                binary_secret: None,
            }),
            error: None,
        };

        let response_xkey = XKey::new();

        let encrypted = Self::encrypt_response(secret_response, &response_xkey, &host_xkey)?;

        let mut headers = HeaderMap::new();
        headers.insert(RESPONSE_XKEY, response_xkey.public_key().as_str());

        // TODO: handle the potential publish error
        let _ = self
            .nats_client
            .publish_with_headers(reply_to, headers, encrypted.into())
            .await;

        Ok(())
    }

    async fn handle_server_xkey_request(&self, reply_to: Subject) {
        // TODO: handle the potential publish error
        let _ = self
            .nats_client
            .publish(reply_to, self.xkey.public_key().into())
            .await;
    }

    fn extract_host_xkey(message: &async_nats::Message) -> StdResult<XKey, GetSecretError> {
        let wasmcloud_host_xkey = message
            .headers
            .clone()
            .unwrap_or_default()
            .get(WASMCLOUD_HOST_XKEY)
            .map(|key| key.to_string())
            .ok_or_else(|| {
                GetSecretError::Other(format!("missing {WASMCLOUD_HOST_XKEY} header"))
            })?;

        XKey::from_public_key(&wasmcloud_host_xkey).map_err(|_| GetSecretError::InvalidXKey)
    }

    fn extract_secret_request(
        message: &async_nats::Message,
        recipient: &XKey,
        sender: &XKey,
    ) -> StdResult<SecretRequest, GetSecretError> {
        let payload = recipient
            .open(&message.payload, sender)
            .map_err(|_| GetSecretError::DecryptionError)?;

        serde_json::from_slice::<SecretRequest>(&payload)
            .map_err(|_| GetSecretError::Other("unable to deserialize the request".to_string()))
    }

    fn validate_and_extract_claims(
        request: &SecretRequest,
    ) -> StdResult<RequestClaims, GetSecretError> {
        // Ensure we have valid claims before we attempt to use them to fetch secrets.
        request
            .context
            .valid_claims()
            .map_err(|e| GetSecretError::InvalidEntityJWT(e.to_string()))?;

        RequestClaims::try_from(request)
            .map_err(|e| GetSecretError::InvalidEntityJWT(e.to_string()))
    }

    fn validate_and_extract_policy(
        request: &SecretRequest,
    ) -> StdResult<VaultPolicy, GetSecretError> {
        VaultPolicy::try_from(request).map_err(|e| GetSecretError::Other(e.to_string()))
    }

    fn convert_nkey_to_encoding_key(nkey: &KeyPair) -> StdResult<EncodingKey, GetSecretError> {
        let seed = nkey
            .seed()
            .map_err(|_| GetSecretError::Other("failed to access nkey seed".to_string()))?;

        let (_prefix, seed_bytes) = nkeys::decode_seed(&seed)
            .map_err(|_| GetSecretError::Other("unable to decode nkey seed".to_string()))?;

        let secret_document = ed25519_dalek::SigningKey::from_bytes(&seed_bytes)
            .to_pkcs8_der()
            .map_err(|_| {
                GetSecretError::Other("failed to generate signing for encoding".to_string())
            })?;

        Ok(EncodingKey::from_ed_der(secret_document.as_bytes()))
    }

    fn encode_claims_to_jwt(
        claims: VaultAuthClaims,
        encoding_key: &EncodingKey,
    ) -> StdResult<String, GetSecretError> {
        jsonwebtoken::encode(
            &jsonwebtoken::Header::new(Algorithm::EdDSA),
            &claims,
            encoding_key,
        )
        .map_err(|_| GetSecretError::Other("failed to encode claims to jwt".to_string()))
    }

    async fn authenticate_with_vault(
        config: &VaultConfig,
        policy: &VaultPolicy,
        jwt: &str,
    ) -> StdResult<VaultClient, GetSecretError> {
        let namespace = policy
            .namespace
            .clone()
            .or(config.default_namespace.clone());
        let settings = VaultClientSettingsBuilder::default()
            .address(config.address.clone())
            .namespace(namespace)
            .build()
            .map_err(|_| {
                GetSecretError::Other("failed to initialize vault client settings".into())
            })?;

        let mut client = VaultClient::new(settings)
            .map_err(|_| GetSecretError::Other("failed to initialize Vault client".into()))?;

        // Authenticate against Vault
        let auth = vaultrs::auth::oidc::login(
            &client,
            &config.auth_mount,
            jwt,
            Some(policy.role_name.clone()),
        )
        .await
        .map_err(|e| GetSecretError::UpstreamError(e.to_string()))?;

        // Use the returned token
        client.set_token(&auth.client_token);

        Ok(client)
    }

    async fn fetch_secret(
        client: &VaultClient,
        mount: &str,
        secret_ref: &VaultSecretRef,
    ) -> Result<ReadSecretResponse, GetSecretError> {
        let request = ReadSecretRequest::builder()
            .mount(mount)
            .path(&secret_ref.path)
            .version(secret_ref.version)
            .build()
            .unwrap();

        vaultrs::api::exec_with_result(client, request)
            .await
            .map_err(|e| GetSecretError::UpstreamError(e.to_string()))
    }

    fn encrypt_response(
        response: SecretResponse,
        sender: &XKey,
        recipient: &XKey,
    ) -> Result<Vec<u8>, GetSecretError> {
        let encoded = serde_json::to_vec(&response)
            .map_err(|_| GetSecretError::Other("unable to encode secret response".to_string()))?;

        sender
            .seal(&encoded, recipient)
            .map_err(|_| GetSecretError::Other("unable to encrypt secret response".to_string()))
    }
}

fn parse_op_from_subject(subject: &str, subject_prefix: &str) -> Result<Operation, GetSecretError> {
    let partial = subject
        .trim_start_matches(subject_prefix)
        .trim_start_matches('.')
        .split('.')
        .collect::<Vec<_>>();

    if partial.len() > 1 {
        return Err(GetSecretError::InvalidRequest);
    }

    partial[0]
        .parse()
        .map_err(|_| GetSecretError::InvalidRequest)
}
