# Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
# See LICENSE.txt for license information.
FROM golang:1.17
WORKDIR /awat/
COPY . /awat/
RUN make build
ENTRYPOINT ["/usr/bin/make", "e2e"]
