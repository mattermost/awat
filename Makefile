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

test: build
	@echo Running tests
	go test ./...

test-image:
	docker build -f test/Dockerfile -t mattermost/awat-e2e .

mocks:
	mockgen -source ./internal/api/store.go Store -package mocks > ./internal/mocks/api/store.go

e2e: e2e-warn e2e-mattermost e2e-slack e2e-parallel

e2e-slack:
	go test -v -tags e2e -count 1 ./test/e2e -timeout 30m -run Slack

e2e-mattermost:
	go test -v -tags e2e -count 1 ./test/e2e -timeout 15m -run Mattermost

e2e-parallel:
	go test -v -tags e2e -count 1 ./test/e2e -timeout 30m -run TwoInQuickSuccession

e2e-warn:
	@echo Warning!
	@echo These tests won\'t work without the following environment variables:
	@echo AWAT_E2E_INSTALLATION_DOMAIN set to the domain to use for testing installations, e.g. \".dev.cloud.mattermost.com\"
	@echo AWAT_E2E_URL set to the AWAT endpoint
	@echo AWAT_E2E_PROVISIONER_URL set to the Provisioner endpoint
	@echo AWAT_E2E_BUCKET set to the address of the S3 bucket

.PHONY: release
release:
	@echo Cut a release
	bash ./scripts/release.sh

.PHONY: deps
deps:
	sudo apt update && sudo apt install hub git
	go get k8s.io/release/cmd/release-notes