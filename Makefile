# Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
# See LICENSE.txt for license information.

################################################################################
##                             VERSION PARAMS                                 ##
################################################################################

## Docker Build Versions
DOCKER_BUILD_IMAGE = golang:1.20
DOCKER_BASE_IMAGE = alpine:3.19

################################################################################

AWAT_IMAGE ?= mattermost/awat:test
AWAT_IMAGE_REPO ?= mattermost/awat

BUILD_HASH = $(shell git rev-parse HEAD)
LDFLAGS += -X "model.BuildHash=$(BUILD_HASH)"
ARCH ?= amd64

GO ?= $(shell command -v go 2> /dev/null)

################################################################################


all: dist

dist: build

.PHONY: build
build: ## build the AWAT
	@echo Building AWAT for ARCH=$(ARCH)
	@if [ "$(ARCH)" = "amd64" ]; then \
		export GOARCH="amd64"; \
	elif [ "$(ARCH)" = "arm64" ]; then \
		export GOARCH="arm64"; \
	elif [ "$(ARCH)" = "arm" ]; then \
		export GOARCH="arm"; \
	else \
		echo "Unknown architecture $(ARCH)"; \
		exit 1; \
	fi; \
	GOOS=linux CGO_ENABLED=0 $(GO) build -ldflags '$(LDFLAGS)' -gcflags all=-trimpath=$(PWD) -asmflags all=-trimpath=$(PWD) -a -installsuffix cgo -o build/_output/bin/awat-$$GOARCH  ./cmd/awat

.PHONY: build-image
build-image:   ## Build the docker image for the AWAT
	@echo Building AWAT Docker Image
	@if [ -z "$(DOCKER_USERNAME)" ] || [ -z "$(DOCKER_PASSWORD)" ]; then \
		echo "DOCKER_USERNAME and/or DOCKER_PASSWORD not set. Skipping Docker login."; \
	else \
		echo $(DOCKER_PASSWORD) | docker login --username $(DOCKER_USERNAME) --password-stdin; \
	fi
	docker buildx build \
	--platform linux/arm64,linux/amd64 \
	--build-arg DOCKER_BUILD_IMAGE=$(DOCKER_BUILD_IMAGE) \
	--build-arg DOCKER_BASE_IMAGE=$(DOCKER_BASE_IMAGE) \
	. -f build/Dockerfile -t $(AWAT_IMAGE) \
	--no-cache \
	--push

.PHONY: build-image-with-tag
build-image-with-tag:   ## Build the docker image for the AWAT
	@echo Building AWAT Docker Image
	@if [ -z "$(DOCKER_USERNAME)" ] || [ -z "$(DOCKER_PASSWORD)" ]; then \
		echo "DOCKER_USERNAME and/or DOCKER_PASSWORD not set. Skipping Docker login."; \
	else \
		echo $(DOCKER_PASSWORD) | docker login --username $(DOCKER_USERNAME) --password-stdin; \
	fi
	docker buildx build \
	--platform linux/arm64,linux/amd64 \
	--build-arg DOCKER_BUILD_IMAGE=$(DOCKER_BUILD_IMAGE) \
	--build-arg DOCKER_BASE_IMAGE=$(DOCKER_BASE_IMAGE) \
	. -f build/Dockerfile -t $(AWAT_IMAGE) -t $(AWAT_IMAGE_REPO):${TAG} \
	--no-cache \
	--push

.PHONY: push-image-pr
push-image-pr:
	@echo Push Image PR
	./scripts/push-image-pr.sh

.PHONY: push-image
push-image:
	@echo Push Image
	./scripts/push-image.sh

.PHONY:check-boilerplate
push-image:
	@echo Check boilerplate
	./scripts/verify-boilerplate.sh

.PHONY: test
test: build
	@echo Running tests
	$(GO) test ./...

.PHONY: test-image
test-image:
	docker build -f test/Dockerfile -t mattermost/awat-e2e .

.PHONY: mocks
mocks:
	mockgen -source ./internal/api/store.go Store -package mocks > ./internal/mocks/api/store.go

.PHONY: e2e
e2e: e2e-warn e2e-mattermost e2e-slack e2e-parallel

.PHONY: e2e-slack
e2e-slack:
	$(GO) test -v -tags e2e -count 1 ./test/e2e -timeout 30m -run Slack

.PHONY: e2e-mattermost
e2e-mattermost:
	$(GO) test -v -tags e2e -count 1 ./test/e2e -timeout 15m -run Mattermost

.PHONY: e2e-parallel
e2e-parallel:
	$(GO) test -v -tags e2e -count 1 ./test/e2e -timeout 30m -run TwoInQuickSuccession

.PHONY: e2e-warn
e2e-warn:
	@echo Warning!
	@echo "These tests won\'t work without the following environment variables:"
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
	$(GO) install k8s.io/release/cmd/release-notes@latest
