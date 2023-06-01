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

VERSION ?= $(shell git describe --tags --always)
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
# Acceptance cluster Create/Delete/Prepare

acceptance-cluster-delete:
	k3d cluster delete s3gw-acceptance
	@if test -f /usr/local/bin/rke2-uninstall.sh; then sudo sh /usr/local/bin/rke2-uninstall.sh; fi

acceptance-cluster-create:
	@./scripts/cluster-create.sh
	k3d kubeconfig merge -ad
	kubectl config use-context k3d-s3gw-acceptance

acceptance-cluster-prepare:
	@./scripts/cluster-prepare.sh

acceptance-cluster-s3gw-0-deploy:
	@./scripts/cluster-s3gw-deploy.sh

acceptance-cluster-s3gw-0-undeploy:
	helm uninstall -n s3gw-acceptance-0 s3gw-0

acceptance-context-set:
	k3d kubeconfig merge -ad
	kubectl config use-context k3d-s3gw-acceptance

########################################################################
# Acceptance Tests

FLAKE_ATTEMPTS ?= 2
GINKGO_NODES ?= 1
GINKGO_POLL_PROGRESS_AFTER ?= 200s
STANDARD_TEST_OPTIONS= -v --nodes ${GINKGO_NODES} --poll-progress-after ${GINKGO_POLL_PROGRESS_AFTER} --randomize-all --flake-attempts=${FLAKE_ATTEMPTS} --fail-on-pending

acceptance-test-install:
	ginkgo ${STANDARD_TEST_OPTIONS} acceptance/install
