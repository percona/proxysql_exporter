# Copyright 2015 The Prometheus Authors
# Copyright 2017 Percona LLC
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GO           := go
FIRST_GOPATH := $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))
PROMU        := bin/promu -v

PREFIX              ?= $(shell pwd)
BIN_DIR             ?= $(shell pwd)
DOCKER_IMAGE_NAME   ?= proxysql-exporter
DOCKER_IMAGE_TAG    ?= $(subst /,-,$(shell git rev-parse --abbrev-ref HEAD))
PKGS         				?= ./...

# Race detector is only supported on amd64.
RACE := $(shell test $$(go env GOARCH) != "amd64" || (echo "-race"))

GO_BUILD_LDFLAGS = -X github.com/prometheus/common/version.Version=$(shell cat VERSION) -X github.com/prometheus/common/version.Revision=$(shell git rev-parse HEAD) -X github.com/prometheus/common/version.Branch=$(shell git describe --always --contains --all) -X github.com/prometheus/common/version.BuildUser= -X github.com/prometheus/common/version.BuildDate=$(shell date +%FT%T%z) -s -w

export PMM_RELEASE_PATH?=.

all: format build test

style:
	@echo ">> checking code style"
	@! gofmt -d $(shell find . -path ./... -prune -o -name '*.go' -print) | grep '^'

test:
	@echo ">> running tests"
	@$(GO) test -v -short $(RACE) -coverprofile coverage.txt $(PKGS)

testall:
	@echo ">> running all tests"
	@$(GO) test -v $(RACE) -coverprofile coverage.txt $(PKGS)

format:
	@echo ">> formatting code"
	@$(GO) fmt $(PKGS)

vet:
	@echo ">> vetting code"
	@$(GO) vet $(PKGS)

build: promu
	@echo ">> building binaries"
	@$(PROMU) build --prefix $(PREFIX)

tarball: promu
	@echo ">> building release tarball"
	@$(PROMU) tarball --prefix $(PREFIX) $(BIN_DIR)

docker:
	@echo ">> building docker image"
	@docker build -t "$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)" .

promu:
	@GOOS=$(shell uname -s | tr A-Z a-z) \
		GOARCH=$(subst x86_64,amd64,$(patsubst i%86,386,$(subst aarch64,arm64,$(shell uname -m)))) \
		$(GO) build -modfile=go.mod -o bin/promu github.com/prometheus/promu

release:
	go build -ldflags="$(GO_BUILD_LDFLAGS)" -o $(PMM_RELEASE_PATH)/proxysql_exporter

.PHONY: all style format build test vet tarball docker promu
