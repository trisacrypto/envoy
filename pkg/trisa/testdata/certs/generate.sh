#!/bin/bash

# Remove old key if it exists
keyFiles=("alice.vaspbot.net.pem" "client.trisatest.dev.pem" "trisatest.dev.pem")
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
    -addext "subjectAltName=DNS:alice.vaspbot.net,DNS:*.alice.vaspbot.net,DNS:bufnet"

openssl req -new -newkey rsa:4096 \
    -nodes -keyout client.key.pem -out client.csr \
    -subj "/C=US/ST=California/L=Menlo Park/O=TRISA/OU=Testing/CN=client.trisatest.dev" \
    -addext "subjectAltName=DNS:client.trisatest.dev,DNS:*.client.trisatest.dev,DNS:bufnet"

# Create signed certificates with CA
openssl x509 -req -days 10950 \
    -CA ca.crt -CAkey ca.key \
    -in alice.csr -out alice.vaspbot.net.pem \
    -copy_extensions copyall

openssl x509 -req -days 10950 \
    -CA ca.crt -CAkey ca.key \
    -in client.csr -out client.trisatest.dev.pem \
    -copy_extensions copyall

# Combine files into a single certificate chain
cat ca.crt >> alice.vaspbot.net.pem
cat alice.key.pem >> alice.vaspbot.net.pem
cat ca.crt >> client.trisatest.dev.pem
cat client.key.pem >> client.trisatest.dev.pem
mv ca.crt trisatest.dev.pem

# Cleanup
rm alice.csr alice.key.pem
rm client.csr client.key.pem
rm ca.key