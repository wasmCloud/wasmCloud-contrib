// Generated by `wit-bindgen` 0.28.0. DO NOT EDIT!
fn wasmImportNow() -> Int64 = "wasi:clocks/monotonic-clock@0.2.0" "now"

/// Read the current value of the clock.
///
/// The clock is monotonic, therefore calling this function repeatedly will
/// produce a sequence of non-decreasing values.
pub fn now() -> UInt64 {
  let result : Int64 = wasmImportNow()
  return result.to_uint64()
}

fn wasmImportResolution() -> Int64 = "wasi:clocks/monotonic-clock@0.2.0" "resolution"

/// Query the resolution of the clock. Returns the duration of time
/// corresponding to a clock tick.
pub fn resolution() -> UInt64 {
  let result : Int64 = wasmImportResolution()
  return result.to_uint64()
}

fn wasmImportSubscribeInstant(p0 : Int64) -> Int = "wasi:clocks/monotonic-clock@0.2.0" "subscribe-instant"

/// Create a `pollable` which will resolve once the specified instant
/// has occurred.
pub fn subscribe_instant(when : UInt64) -> @poll.Pollable {
  let result : Int = wasmImportSubscribeInstant(when.to_int64())
  return @poll.Pollable::Pollable(result)
}

fn wasmImportSubscribeDuration(p0 : Int64) -> Int = "wasi:clocks/monotonic-clock@0.2.0" "subscribe-duration"

/// Create a `pollable` that will resolve after the specified duration has
/// elapsed from the time this function is invoked.
pub fn subscribe_duration(when : UInt64) -> @poll.Pollable {
  let result : Int = wasmImportSubscribeDuration(when.to_int64())
  return @poll.Pollable::Pollable(result)
}
