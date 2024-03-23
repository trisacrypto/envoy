#!/bin/bash

# Remove old key if it exists
if [ -f trisa.example.dev.pem ]; then
    rm localhost.pem
fi

# Create CA
openssl req -x509 -newkey rsa:4096 -sha256 -days 10950 \
    -nodes -keyout ca.key -out ca.crt \
    -subj "/C=US/ST=California/L=Menlo Park/O=TRISA/OU=TestNet/CN=trisatest.dev" \
    -addext "subjectAltName=DNS:trisatest.dev,DNS:*.trisatest.dev"

# Create certificate request
openssl req -newkey rsa:4096 \
    -nodes -keyout key.pem -out example.csr \
    -subj "/C=US/ST=Maryland/L=Queenstown/O=Rotational/OU=Testing/CN=trisa.example.dev" \
    -addext "subjectAltName=DNS:trisa.example.dev,DNS:*.trisa.example.dev"

# Create signed certificates with CA
openssl x509 -req -days 10950 \
    -CA ca.crt -CAkey ca.key \
    -in example.csr -out trisa.example.dev.pem

# Combine files into a single certificate chain
cat ca.crt >> trisa.example.dev.pem
cat key.pem >> trisa.example.dev.pem

# Cleanup
rm example.csr
rm key.pem
rm ca.key
rm ca.crt