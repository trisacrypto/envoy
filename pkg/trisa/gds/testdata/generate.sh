#!/bin/bash

# Remove old key if it exists
if [ -f localhost.pem ]; then
    rm localhost.pem
fi

# Create CA
openssl req -x509 -newkey rsa:4096 -sha256 -days 10950 \
    -nodes -keyout ca.key -out ca.crt \
    -subj "/C=US/ST=California/L=Menlo Park/O=TRISA/OU=TestNet/CN=trisatest.dev" \
    -addext "subjectAltName=DNS:trisatest.dev,DNS:*.trisatest.dev"

# Create certificate request
openssl req -newkey rsa:4096 \
    -nodes -keyout local.key.pem -out localhost.csr \
    -subj "/C=US/ST=Maryland/L=Queenstown/O=Rotational/OU=Testing/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,DNS:*.localhost,IP:127.0.0.1"

# Create signed certificates with CA
openssl x509 -req -days 10950 \
    -CA ca.crt -CAkey ca.key \
    -in localhost.csr -out localhost.pem

# Combine files into a single certificate chain
cat ca.crt >> localhost.pem
cat local.key.pem >> localhost.pem

# Cleanup
rm localhost.csr
rm local.key.pem
rm ca.key
rm ca.crt