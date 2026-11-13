#!/bin/bash
set -exuo pipefail

docker stop lab1 || true

docker build -t lab1:1.0 .
docker run --rm -d \
  --name lab1 \
  -p 127.0.0.1:8080:8080 \
  -e LOG_LEVEL=debug \
  lab1:1.0
