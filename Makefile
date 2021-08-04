# Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
# See LICENSE.txt for license information.

AWAT_IMAGE ?= mattermost/awat:test

BUILD_HASH = $(shell git rev-parse HEAD)
LDFLAGS += -X "model.BuildHash=$(BUILD_HASH)"


GO ?= $(shell command -v go 2> /dev/null)

all: dist

dist: build

.PHONY: build
build: ## build the AWAT
	@echo Building AWAT
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -ldflags '$(LDFLAGS)' -gcflags all=-trimpath=$(PWD) -asmflags all=-trimpath=$(PWD) -a -installsuffix cgo -o build/_output/bin/awat  ./cmd/awat

build-image:   ## Build the docker image for the AWAT
	@echo Building AWAT Docker Image
	docker build \
	. -f build/Dockerfile -t $(AWAT_IMAGE) 
# --no-cache

test: build
	@echo Running tests
	go test ./...

mocks:
	mockgen -source ./internal/api/store.go Store -package mocks > ./internal/mocks/api/store.go

e2e:
	@echo Warning!
	@echo These tests won\'t work without the following environment variables pointing to prepared services:
	@echo PROVISIONER_URL to point to where the Provisioner is listening
	@echo AWAT_BUCKET to the address of the S3 bucket
	@echo AWAT_DATABASE to the address of the Postgres instance
	go test -tags e2e ./test/e2e -timeout 15m
