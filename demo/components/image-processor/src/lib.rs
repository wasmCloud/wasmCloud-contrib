wit_bindgen::generate!({ generate_all });
use crate::wasi::logging::logging::{log, Level};
use crate::wasmcloud::task_manager::tracker;
use bytes::Bytes;
use exports::wasmcloud::image_processor::resizer::Guest;
use image::ImageReader;
use std::io::Cursor;
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
    // wrap the operation into a task
    fn resize(body: Vec<u8>, width: u32, height: u32) -> Result<String, String> {
        let operation = tracker::start("resize", "some asset").map_err(|e| e.to_string())?;

        let res = ImageResizer::do_resize(&operation, body, width, height);

        match &res {
            Ok(resized) => {
                tracker::complete(&operation, resized).map_err(|e| e.to_string())?;
            }
            Err(e) => {
                tracker::fail(&operation, e).map_err(|e| e.to_string())?;
            }
        }

        res
    }

    fn serve(asset: String) -> Result<Vec<u8>, String> {
        match objstore::read_object(BUCKET_NAME, &asset) {
            Ok(bytes) => Ok(bytes.to_vec()),
            Err(e) => Err(e.to_string()),
        }
    }
}

impl ImageResizer {
    fn do_resize(
        asset_key: &String,
        body: Vec<u8>,
        width: u32,
        height: u32,
    ) -> Result<String, String> {
        objstore::ensure_container(&BUCKET_NAME.to_string()).map_err(|e| e.to_string())?;

        let original_image_key = format!("original-{}", asset_key);
        objstore::write_object(Bytes::from(body.clone()), BUCKET_NAME, &original_image_key)
            .map_err(|e| e.to_string())?;

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
