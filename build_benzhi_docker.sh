#!/bin/bash
set -e

IMAGE_NAME=${1:-lan-clipboard-sync}
DOCKER_PLATFORM=${2:-linux/amd64}

docker build --platform "$DOCKER_PLATFORM" -f benzhi.Dockerfile -t "$IMAGE_NAME" .

echo "Docker image '$IMAGE_NAME' built successfully!"
echo "Interactive shell: docker run -it $IMAGE_NAME:latest"
