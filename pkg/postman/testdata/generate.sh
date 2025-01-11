#!/bin/bash

# Remove old key if it exists
if [ -f "certificate.pem" ]; then
    rm "certificate.pem"
fi

# Generate a new self-signed certificate with a private key
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out "certificate.pem" -days 3650 -nodes -subj "/CN=testing"

cat key.pem >> "certificate.pem"
rm key.pem