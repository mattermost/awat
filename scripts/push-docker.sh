#!/bin/bash
set -e
set -u

if [ -z "${CIRCLE_TAG:-}" ]; then
  echo "Pushing latest for $CIRCLE_BRANCH..."
  export TAG=latest
else
  echo "Pushing release $CIRCLE_TAG..."
  export TAG="$CIRCLE_TAG"
fi
echo $DOCKER_PASSWORD | docker login --username $DOCKER_USERNAME --password-stdin

docker tag mattermost/awat:test mattermost/awat:$TAG
docker tag mattermost/awat-e2e:latest mattermost/awat-e2e:$TAG

docker push mattermost/awat:$TAG
docker push mattermost/awat-e2e:$TAG