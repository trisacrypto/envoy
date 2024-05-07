# Dynamic Builds
ARG XX_IMAGE=tonistiigi/xx
ARG BUILDER_IMAGE=golang:1.21-bullseye
ARG FINAL_IMAGE=debian:bullseye-slim

# Build stage
FROM --platform=${BUILDPLATFORM} ${XX_IMAGE} AS xx
FROM --platform=${BUILDPLATFORM} ${BUILDER_IMAGE} AS builder

# Copy XX scripts to the build stage
COPY --from=xx / /

# Build Args
ARG GIT_REVISION=""

# Platform args
ARG TARGETOS
ARG TARGETARCH
ARG TARGETPLATFORM

# Ensure ca-certificates are up to date
RUN update-ca-certificates

# Prepare for cross-compilation
RUN apt-get update && apt-get install -y clang lld
RUN xx-apt install -y libc6-dev

# Use modules for dependencies
WORKDIR $GOPATH/src/github.com/trisacrypto/envoy

COPY go.mod .
COPY go.sum .

ENV CGO_ENABLED=1
ENV GO111MODULE=on
RUN go mod download
RUN go mod verify

# Copy package
COPY . .

# Build binary
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} xx-go build -o /go/bin/envoy -ldflags="-X 'github.com/trisacrypto/envoy/pkg.GitVersion=${GIT_REVISION}'" ./cmd/envoy && xx-verify /go/bin/envoy

# Final Stage
FROM --platform=${BUILDPLATFORM} ${FINAL_IMAGE} AS final

LABEL maintainer="TRISA <info@trisa.io>"
LABEL description="TRISA Self Hosted Node"

# Ensure ca-certificates are up to date
RUN set -x && apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates sqlite3 && \
    rm -rf /var/lib/apt/lists/*

# Copy the binary to the production image from the builder stage
COPY --from=builder /go/bin/envoy /usr/local/bin/envoy

CMD [ "/usr/local/bin/envoy", "serve" ]