#!/bin/bash

openssl req -x509 -newkey rsa:4096 -sha256 -days 10950 \
    -nodes -keyout local.key.pem -out local.pem \
    -subj "/C=US/ST=Maryland/L=Queenstown/O=Rotational/OU=Engineering/CN=alpha.trisa.dev" \
    -addext "subjectAltName=DNS:alpha.trisa.dev,DNS:*.alpha.trisa.dev,IP:127.0.0.1"

cat local.key.pem >> local.pem
rm local.key.pem

openssl req -x509 -newkey rsa:4096 -sha256 -days 10950 \
    -nodes -out remote.pem \
    -subj "/C=US/ST=Minnesota/L=Minneapolis/O=Rotational/OU=Counterparty/CN=bravo.trisa.dev" \
    -addext "subjectAltName=DNS:bravo.trisa.dev,DNS:*.bravo.trisa.dev,IP:127.0.0.1"