FROM golang:1.22 AS builder
ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /workspace
COPY go.* .
RUN go mod download

COPY *.go .
COPY pkg/ pkg/

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -o secrets-kubernetes

FROM gcr.io/distroless/static-debian12:debug AS debug
COPY --from=builder /workspace/secrets-kubernetes .
USER 65532:65532
ENTRYPOINT ["/secrets-kubernetes"]

FROM gcr.io/distroless/static-debian12:nonroot AS release
WORKDIR /
COPY --from=builder /workspace/secrets-kubernetes .
USER 65532:65532

ENTRYPOINT ["/secrets-kubernetes"]
