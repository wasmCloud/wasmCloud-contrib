wit_bindgen::generate!({ generate_all });

use crate::wasi::config::runtime::get;
use crate::wasi::http::outgoing_handler;
use crate::wasi::http::outgoing_handler::OutgoingRequest;
use crate::wasi::http::types::{Fields, Scheme};
use crate::wasi::logging::logging::{log, Level};
use anyhow::{anyhow, bail, Context as _, Result};
use base64::{engine::general_purpose, Engine as _};
use bytes::{Bytes, BytesMut};
use exports::wasmcloud::image_analyzer::analyzer::Guest;
use wasi::http::types::{Method, OutgoingBody};
use wasi::io::streams::StreamError;

const MAX_READ_BYTES: usize = 2 * 1024 * 1024;

struct Ollama;

#[derive(serde::Serialize)]
struct OllamaRequest {
    model: String,
    prompt: String,
    stream: bool,
    images: Vec<String>,
}

#[derive(serde::Deserialize)]
struct OllamaResponse {
    response: String,
}

impl Guest for Ollama {
    fn detect(image: Vec<u8>) -> Result<bool, String> {
        let endpoint = match crate::wasi::config::runtime::get("endpoint") {
            Ok(Some(addr)) => addr,
            Ok(None) => "127.0.0.1:11434".to_string(),
            Err(_) => return Err("Failed to get endpoint config".to_string()),
        };

        let prompt = match crate::wasi::config::runtime::get("prompt") {
            Ok(Some(p)) => p,
            Ok(None) => {
                "Answer with true or false and nothing else. Does this image contain an animal?"
                    .to_string()
            }
            Err(_) => return Err("Failed to get prompt config".to_string()),
        };

        let model = match crate::wasi::config::runtime::get("model") {
            Ok(Some(p)) => p,
            Ok(None) => "llava".to_string(),
            Err(_) => return Err("Failed to get model config".to_string()),
        };

        let positive_response = match crate::wasi::config::runtime::get("positive_response") {
            Ok(Some(p)) => p,
            Ok(None) => "Yes".to_string(),
            Err(_) => return Err("Failed to get positive_response config".to_string()),
        };

        let req = OutgoingRequest::new(Fields::new());
        req.set_scheme(Some(&Scheme::Http))
            .map_err(|()| "failed to set scheme")?;
        req.set_authority(Some(&endpoint))
            .map_err(|()| "failed to set authority")?;
        req.set_path_with_query(Some("/api/generate"))
            .map_err(|()| "failed to set path and query")?;
        req.set_method(&Method::Post)
            .map_err(|()| "failed to set method")?;

        let encoded_image = general_purpose::STANDARD.encode(image);

        let contents = OllamaRequest {
            prompt,
            model,
            stream: false,
            images: vec![encoded_image],
        };

        let contents = serde_json::to_vec(&contents).unwrap();

        let resp = req.fetch_bytes(contents).map_err(|e| e.to_string())?;

        let llm_resp: OllamaResponse = serde_json::from_slice(&resp).map_err(|e| e.to_string())?;

        Ok(llm_resp.response.trim() == positive_response)
    }
}

impl OutgoingRequest {
    fn fetch_bytes(self, payload: Vec<u8>) -> Result<Bytes> {
        let body_writer = self.body().unwrap();

        let resp =
            outgoing_handler::handle(self, None).map_err(|e| anyhow!("request failed: {e}"))?;

        let body_stream = body_writer.write().unwrap();
        for chunk in payload.chunks(1024) {
            body_stream
                .blocking_write_and_flush(chunk)
                .expect("failed to write body");
        }

        drop(body_stream);
        OutgoingBody::finish(body_writer, None).expect("failed to finish body");

        resp.subscribe().block();
        let response = resp
            .get()
            .context("HTTP request response missing")?
            .map_err(|()| anyhow!("HTTP request response requested more than once"))?
            .map_err(|code| anyhow!("HTTP request failed (error code {code})"))?;

        if response.status() != 200 {
            bail!("response failed, status code [{}]", response.status());
        }

        let response_body = response
            .consume()
            .map_err(|()| anyhow!("failed to get incoming request body"))?;

        let mut buf = BytesMut::with_capacity(MAX_READ_BYTES);
        let stream = response_body
            .stream()
            .expect("failed to get HTTP request response stream");
        loop {
            match stream.blocking_read(MAX_READ_BYTES as u64) {
                Ok(bytes) if bytes.is_empty() => {
                    break;
                }
                Ok(bytes) => {
                    buf.extend(bytes);
                }
                Err(StreamError::Closed) => {
                    break;
                }
                Err(e) => {
                    bail!("failed to read bytes: {e}")
                }
            }
        }

        Ok(buf.freeze())
    }
}

export!(Ollama);
