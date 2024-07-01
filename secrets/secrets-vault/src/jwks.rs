use anyhow::Result;
use axum::{extract::State, http::StatusCode, routing::get, Json, Router};
use nkeys::KeyPair;
use crate::jwk::JsonWebKey;
use serde::{Deserialize, Serialize};
use std::{net::SocketAddrV4, sync::Arc};

#[derive(Debug)]
pub(crate) struct VaultSecretsJwksServer {
    keys: Vec<JsonWebKey>,
    listen_address: SocketAddrV4,
}

struct SharedState {
    keys: Vec<JsonWebKey>,
}

#[derive(Debug, Serialize, Deserialize)]
struct JwksResponse {
    keys: Vec<JsonWebKey>,
}

impl VaultSecretsJwksServer {
    pub fn new(nkeys: Vec<KeyPair>, listen_address: SocketAddrV4) -> Result<Self> {
        let mut keys = vec![];
        for kp in nkeys {
            keys.push(JsonWebKey::try_from(kp)?);
        }
        Ok(Self {
            keys,
            listen_address,
        })
    }

    pub async fn serve(&self) -> Result<()> {
        let state = Arc::new(SharedState {
            keys: self.keys.clone(),
        });
        let app = Router::new()
            .route("/.well-known/keys", get(handle_well_known_keys))
            .with_state(state);

        let listener = tokio::net::TcpListener::bind(self.listen_address).await?;
        axum::serve(listener, app).await?;

        Ok(())
    }
}

async fn handle_well_known_keys(
    State(state): State<Arc<SharedState>>,
) -> (StatusCode, Json<JwksResponse>) {
    (
        StatusCode::OK,
        Json(JwksResponse {
            keys: state.keys.clone(),
        }),
    )
}
