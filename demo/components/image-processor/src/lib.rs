wit_bindgen::generate!({ generate_all });
use crate::wasi::logging::logging::{log, Level};
use bytes::Bytes;
use exports::wasmcloud::image_processor::resizer::Guest;
use image::ImageReader;
use std::io::Cursor;
use uuid::Uuid;
use wasi::blobstore::blobstore;

mod objstore;

/// Maximum bytes to write at a time, due to the limitations on wasi-io's blocking_write_and_flush()
const MAX_WRITE_BYTES: usize = 4096;

/// Maximum bytes to read at a time from the incoming request body
/// this value is chosen somewhat arbitrarily, and is not a limit for bytes read,
/// but is instead the amount of bytes to be read *at once*
const MAX_READ_BYTES: usize = 2048;

const BUCKET_NAME: &str = "images";

struct ImageResizer;

impl Guest for ImageResizer {
    fn upload(body: Vec<u8>) -> Result<String, String> {
        objstore::ensure_container(&BUCKET_NAME.to_string()).map_err(|e| e.to_string())?;

        let asset_key = Uuid::new_v4().to_string().to_lowercase();

        match objstore::write_object(Bytes::from(body.clone()), BUCKET_NAME, &asset_key) {
            Ok(_) => Ok(asset_key),
            Err(e) => Err(e.to_string()),
        }
    }

    fn serve(asset: String) -> Result<Vec<u8>, String> {
        match objstore::read_object(BUCKET_NAME, &asset) {
            Ok(bytes) => Ok(bytes.to_vec()),
            Err(e) => Err(e.to_string()),
        }
    }

    fn resize(asset_key: String, width: u32, height: u32) -> Result<String, String> {
        objstore::ensure_container(&BUCKET_NAME.to_string()).map_err(|e| e.to_string())?;

        let body = match objstore::read_object(BUCKET_NAME, &asset_key) {
            Ok(bytes) => bytes,
            Err(e) => return Err(e.to_string()),
        };

        let original_image = ImageReader::new(Cursor::new(body))
            .with_guessed_format()
            .map_err(|e| e.to_string())?
            .decode()
            .map_err(|e| e.to_string())?;

        let resized_image =
            original_image.resize(width, height, image::imageops::FilterType::Nearest);

        let mut resized_bytes: Vec<u8> = Vec::new();
        resized_image
            .write_to(
                &mut Cursor::new(&mut resized_bytes),
                image::ImageFormat::Png,
            )
            .map_err(|e| e.to_string())?;

        let resized_image_key = format!("resized-{}", asset_key);
        objstore::write_object(Bytes::from(resized_bytes), BUCKET_NAME, &resized_image_key)
            .map_err(|e| e.to_string())?;

        Ok(resized_image_key)
    }
}

export!(ImageResizer);
