# Dynamic Builds
ARG XX_IMAGE=tonistiigi/xx
ARG BUILDER_IMAGE=golang:1.21-alpine
ARG FINAL_IMAGE=debian:bullseye-slim

# Build stage
FROM --platform=${BUILDPLATFORM} ${XX_IMAGE} AS xx
FROM --platform=${BUILDPLATFORM} ${BUILDER_IMAGE} AS builder

# Add compilers to alpine
RUN apk add clang lld

# Copy XX scripts to the build stage
COPY --from=xx / /

# Build Args
ARG GIT_REVISION=""
ARG TARGETPLATFORM

# Prepare for cross-compilation
RUN xx-apk add musl-dev gcc

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
ARG TARGETOS
ARG TARGETARCH
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} xx-go build -o /go/bin/envoy -ldflags="-X 'github.com/trisacrypto/envoy/pkg.GitVersion=${GIT_REVISION}'" ./cmd/envoy && xx-verify /go/bin/envoy

# Final Stage
FROM --platform=${BUILDPLATFORM} ${FINAL_IMAGE} AS final

LABEL maintainer="TRISA <info@trisa.io>"
LABEL description="TRISA Self Hosted Node"

# Ensure ca-certificates are up to date
RUN set -x && apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates postgresql-client && \
    rm -rf /var/lib/apt/lists/*

# Copy the binary to the production image from the builder stage
COPY --from=builder /go/bin/envoy /usr/local/bin/envoy

CMD [ "/usr/local/bin/envoy", "serve" ]