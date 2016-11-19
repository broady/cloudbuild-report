#!/bin/sh

if [[ -z "$TAG" ]]; then
  echo 'Error: output docker image tag name must be provided as $TAG.' 1>&2
  exit 1
fi
if [[ -z "$BASE" ]]; then
  BASE=debian:latest
fi

set -e -x

tee /workspace/.cacher-Dockerfile <<EOF
FROM $BASE
COPY . /cache
EOF

docker build --tag $TAG -f /workspace/.cacher-Dockerfile /workspace
