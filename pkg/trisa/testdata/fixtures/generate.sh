#!/bin/bash

ENVELOPE_ID="f04d02cb-7b88-44d1-a6c8-ddb6cbbd2d25"

trisa make --envelope-id $ENVELOPE_ID \
    -i payloads/identity.pb.json -t payloads/transaction.pb.json \
    -S $(date -u +"%Y-%m-%dT%H:%M:%SZ") \
    --sealing-key ../certs/alice.vaspbot.com.pem \
    --out secenv_transaction.pb.json

trisa make --envelope-id $ENVELOPE_ID \
    -i payloads/identity.pb.json -t payloads/pending.pb.json \
    -S $(date -u +"%Y-%m-%dT%H:%M:%SZ") \
    --sealing-key ../certs/alice.vaspbot.com.pem \
    --out secenv_pending.pb.json

