# Dynamic Builds
ARG XX_IMAGE=tonistiigi/xx
ARG BUILDER_IMAGE=dhi.io/golang:1.26-debian13-dev
ARG FINAL_IMAGE=dhi.io/golang:1.26-debian13

# Build stage
FROM --platform=${BUILDPLATFORM} ${XX_IMAGE} AS xx
FROM --platform=${BUILDPLATFORM} ${BUILDER_IMAGE} AS builder

# Copy XX scripts to the build stage
COPY --from=xx / /

# Build Args
ARG GIT_REVISION=""
ARG BUILD_DATE=""

# Platform args
ARG TARGETOS
ARG TARGETARCH
ARG TARGETPLATFORM

# Ensure ca-certificates are up to date
RUN update-ca-certificates

# Prepare for cross-compilation
RUN apt-get update && apt-get install -y clang lld
RUN xx-apt install -y libc6-dev gcc

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
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} xx-go build -v \
    -ldflags="-X 'github.com/trisacrypto/envoy/pkg.GitVersion=${GIT_REVISION}' -X 'github.com/trisacrypto/envoy/pkg.BuildDate=${BUILD_DATE}'" \
    -o /go/bin/envoy \
    ./cmd/envoy && \
    xx-verify /go/bin/envoy

# Final Stage
FROM --platform=${BUILDPLATFORM} ${FINAL_IMAGE} AS final

LABEL maintainer="TRISA <info@travelrule.io>"
LABEL description="TRISA Self Hosted Node"

# Copy the binary to the production image from the builder stage
COPY --from=builder /go/bin/envoy /usr/local/bin/envoy

CMD [ "/usr/local/bin/envoy", "serve" ]
