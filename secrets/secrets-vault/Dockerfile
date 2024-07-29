# syntax=docker/dockerfile:1.4
FROM rust:1.79-slim-bookworm AS builder

WORKDIR /build
COPY . /build

RUN <<EOF
cargo build --release
EOF

# Debug release using the distroless debug image:
# * https://github.com/GoogleContainerTools/distroless/tree/main?tab=readme-ov-file#debug-images
#
# Includes /busybox/shell for debugging, which should be used to override the default entrypoint.
FROM gcr.io/distroless/cc-debian12:debug AS debug
COPY --from=builder /build/target/release/secrets-vault /bin/secrets-vault
ENTRYPOINT ["/bin/secrets-vault"]

FROM gcr.io/distroless/cc-debian12:nonroot AS release
COPY --from=builder /build/target/release/secrets-vault /bin/secrets-vault
ENTRYPOINT ["/bin/secrets-vault"]