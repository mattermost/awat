# Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
# See LICENSE.txt for license information.
ARG DOCKER_BUILD_IMAGE=golang:1.21
ARG DOCKER_BASE_IMAGE=alpine:3.19

FROM ${DOCKER_BUILD_IMAGE} AS build
WORKDIR /awat/
COPY . /awat/

# Detect architecture and set ARCH
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then \
        ARCH="amd64"; \
    elif [ "$ARCH" = "aarch64" ]; then \
        ARCH="arm64"; \
    elif [ "$ARCH" = "armv7l" ] || [ "$ARCH" = "armv6l" ]; then \
        ARCH="arm"; \
    fi && \
    echo "ARCH=$ARCH" && \
    make build ARCH=$ARCH

FROM ${DOCKER_BASE_IMAGE}
LABEL name="Mattermost Cloud Automatic Workspace Archive Translator" \
  maintainer="cloud-team@mattermost.com" \
  vendor="Mattermost" \
  distribution-scope="public" \
  architecture="x86_64, arm64" \
  url="https://mattermost.com" \
  io.k8s.description="Mattermost Cloud AWAT converts workspace archives into a Mattermost-native import format" \
  io.k8s.display-name="Mattermost Cloud AWAT"

COPY --from=build /awat/cmd/awat /usr/local/bin/awat

USER ${USER_UID}

EXPOSE 8099
ENTRYPOINT ["/usr/local/bin/awat"]
