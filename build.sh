#!/bin/bash

set -euo pipefail

SUPPORTED_ARCHS="amd64 arm64"
REGISTRY_URI="${REGISTRY_URI:-887832589804.dkr.ecr.us-west-1.amazonaws.com/image-server}"
APP_REPO_TAG=$REGISTRY_URI:$DOCKER_TAG

# build individual architectures
ALL_ARCH_TAGS=""
for ARCH in $SUPPORTED_ARCHS; do
  echo "*** Building for $ARCH ***"
  TAG=$APP_REPO_TAG-$ARCH
  docker build --pull -t $TAG --platform $ARCH .
  echo "*** Pushing $TAG ***"
  docker push $TAG
  ALL_ARCH_TAGS="$ALL_ARCH_TAGS $TAG"
done

# create multi-arch manifest
echo "*** Creating manifest and pushing $APP_REPO_TAG ***"
docker manifest create $APP_REPO_TAG $ALL_ARCH_TAGS
for ARCH in $SUPPORTED_ARCHS; do
  docker manifest annotate --arch $ARCH $APP_REPO_TAG $APP_REPO_TAG-$ARCH
done

docker manifest push $APP_REPO_TAG
if [ -x "$(command -v buildkite-agent)" ]; then
  buildkite-agent annotate "Image tag: $BGH_RETURNLY_DOCKER_TAG" --style 'info' --context 'build-info'
fi
