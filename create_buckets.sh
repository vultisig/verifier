#!/bin/bash

# wait for minio server
sleep 5

mc alias set local http://localhost:9000 minioadmin minioadmin

buckets="vultisig-verifier"

for bucket in $buckets; do
    mc mb --ignore-existing local/$bucket
done
