# Copyright © 2021 - 2023 SUSE LLC
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#     http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

VERSION ?= $(shell git describe --tags)
CGO_ENABLED ?= 0

tag:
	@git describe --tags --abbrev=0

lint:
	golangci-lint run

tidy:
	go mod tidy

fmt:
	go fmt ./...

########################################################################
# Build

build-images:
	@./scripts/build-images.sh

########################################################################
# Acceptance tests

FLAKE_ATTEMPTS ?= 2
GINKGO_NODES ?= 2
GINKGO_POLL_PROGRESS_AFTER ?= 200s
REGEX ?= ""
STANDARD_TEST_OPTIONS= -v --nodes ${GINKGO_NODES} --poll-progress-after ${GINKGO_POLL_PROGRESS_AFTER} --randomize-all --flake-attempts=${FLAKE_ATTEMPTS} --fail-on-pending

acceptance-cluster-delete:
	k3d cluster delete s3gw-acceptance
	@if test -f /usr/local/bin/rke2-uninstall.sh; then sudo sh /usr/local/bin/rke2-uninstall.sh; fi

acceptance-cluster-setup:
	@./scripts/cluster-setup.sh

acceptance-context-set:
	k3d kubeconfig merge -ad
	kubectl config use-context k3d-s3gw-acceptance

acceptance-environment-prepare:
	@./scripts/prepare-environment.sh
