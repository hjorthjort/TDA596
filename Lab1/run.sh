#!/bin/bash
set -exuo pipefail

NAME=lab1
TAG=latest
DOCKERHUB_USERNAME=lerthsk

docker stop $NAME || true

docker buildx build \
  --platform linux/arm64 \
  -t $DOCKERHUB_USERNAME/$NAME:$TAG \
  ${PUSH:+"--push"} .

if [[ "${LOCAL:-}" == "1" ]]; then
    docker run --rm -d \
        --name $NAME \
        -p 127.0.0.1:8080:8080 \
        -e LOG_LEVEL=debug \
        $NAME:$TAG
fi
