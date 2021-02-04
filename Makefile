# Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
# See LICENSE.txt for license information.

AWAT_IMAGE ?= mattermost/mattermost-cloud-awat:test

BUILD_HASH = $(shell git rev-parse HEAD)
LDFLAGS += -X "model.BuildHash=$(BUILD_HASH)"


GO ?= $(shell command -v go 2> /dev/null)

all: dist

dist: build

.PHONY: build
build: ## build the AWAT
	@echo Building AWAT
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -ldflags '$(LDFLAGS)' -gcflags all=-trimpath=$(PWD) -asmflags all=-trimpath=$(PWD) -a -installsuffix cgo -o build/_output/bin/awat  ./cmd/awat

get-slack-advanced-exporter:
ifeq (,$(wildcard build/bin/slack-advanced-exporter-linux-amd64))
	@echo Getting Slack Advanced Exporter from GitHub
	curl -L https://github.com/grundleborg/slack-advanced-exporter/releases/download/v0.3.0/slack-advanced-exporter-linux-amd64 -o build/bin/slack-advanced-exporter-linux-amd64
endif


build-image:   ## Build the docker image for the AWAT
	@echo Building AWAT Docker Image
	docker build \
	. -f build/Dockerfile -t $(AWAT_IMAGE) 
# --no-cache
