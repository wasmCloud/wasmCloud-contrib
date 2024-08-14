wit_bindgen::generate!({ generate_all });

use crate::wasi::http::outgoing_handler;
use crate::wasi::http::outgoing_handler::OutgoingRequest;
use crate::wasi::http::types::{Fields, Scheme};
use crate::wasi::logging::logging::{log, Level};
use crate::wasmcloud::task_manager::tracker;
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
    fn is_animal(image: Vec<u8>) -> Result<bool, String> {
        let operation = tracker::start("analyze", "some asset").map_err(|e| e.to_string())?;

        let res = Ollama::do_is_animal(image);

        match &res {
            Ok(result) => {
                tracker::complete(&operation, &result.to_string()).map_err(|e| e.to_string())?;
            }
            Err(e) => {
                tracker::fail(&operation, e).map_err(|e| e.to_string())?;
            }
        }

        res
    }
}

impl Ollama {
    fn do_is_animal(image: Vec<u8>) -> Result<bool, String> {
        let req = OutgoingRequest::new(Fields::new());
        req.set_scheme(Some(&Scheme::Http))
            .map_err(|()| "failed to set scheme")?;
        req.set_authority(Some("127.0.0.1:11434"))
            .map_err(|()| "failed to set authority")?;
        req.set_path_with_query(Some("/api/generate"))
            .map_err(|()| "failed to set path and query")?;
        req.set_method(&Method::Post)
            .map_err(|()| "failed to set method")?;

        let encoded_image = general_purpose::STANDARD.encode(image);

        let contents = OllamaRequest {
            model: "llava".to_string(),
            prompt:
                "Answer with true or false and nothing else. Does this image contain an animal?"
                    .to_string(),
            stream: false,
            images: vec![encoded_image],
        };

        let contents = serde_json::to_vec(&contents).unwrap();

        let resp = req.fetch_bytes(contents).map_err(|e| e.to_string())?;

        let llm_resp: OllamaResponse = serde_json::from_slice(&resp).map_err(|e| e.to_string())?;

        Ok(llm_resp.response.trim() == "Yes")
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
