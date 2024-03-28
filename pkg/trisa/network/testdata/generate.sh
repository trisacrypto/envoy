#!/bin/bash

# Remove old key if it exists
keyFiles=("alice.pem" "bob.pem" "pool.pem")
for keyfile in ${keyFiles}[@]}; do
    if [ -f $keyfile ]; then
        rm $keyfile
    fi
done

# Create CA
openssl req -x509 -newkey rsa:4096 -sha256 -days 10950 \
    -nodes -keyout ca.key -out ca.crt \
    -subj "/C=US/ST=California/L=Menlo Park/O=TRISA/OU=TestNet/CN=trisatest.dev" \
    -addext "subjectAltName=DNS:trisatest.dev,DNS:*.trisatest.dev"

# Create certificate requests for alice and bob
openssl req -new -newkey rsa:4096 \
    -nodes -keyout alice.key.pem -out alice.csr \
    -subj "/C=US/ST=New York/L=New York/O=Alice VASP/OU=Testing/CN=alice.vaspbot.net" \
    -addext "subjectAltName=DNS:alice.vaspbot.net,DNS:*.alice.vaspbot.net"

openssl req -new -newkey rsa:4096 \
    -nodes -keyout bob.key.pem -out bob.csr \
    -subj "/C=GB/ST=Oxfordshire/L=Oxford/O=Bob VASP/OU=Testing/CN=bob.vaspbot.net" \
    -addext "subjectAltName=DNS:bob.vaspbot.net,DNS:*.bob.vaspbot.net"

# Create signed certificates with CA
openssl x509 -req -days 10950 \
    -CA ca.crt -CAkey ca.key \
    -in alice.csr -out alice.pem \
    -copy_extensions copyall

openssl x509 -req -days 10950 \
    -CA ca.crt -CAkey ca.key \
    -in bob.csr -out bob.pem \
    -copy_extensions copyall

# Combine files into a single certificate chain
cat ca.crt >> alice.pem
cat alice.key.pem >> alice.pem
cat ca.crt >> bob.pem
cat bob.key.pem >> bob.pem
mv ca.crt pool.pem

# Cleanup
rm alice.csr alice.key.pem
rm bob.csr bob.key.pem
rm ca.key