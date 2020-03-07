# Copyright 2017 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

PKG = sigs.k8s.io/azurefile-csi-driver
GIT_COMMIT ?= $(shell git rev-parse HEAD)
REGISTRY ?= andyzhangx
IMAGE_NAME = azurefile-csi
IMAGE_VERSION ?= v0.5.0
# Use a custom version for E2E tests if we are in Prow
ifdef AZURE_CREDENTIALS
override IMAGE_VERSION := e2e-$(GIT_COMMIT)
endif
IMAGE_TAG = $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION)
IMAGE_TAG_LATEST = $(REGISTRY)/$(IMAGE_NAME):latest
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS ?= "-X ${PKG}/pkg/azurefile.driverVersion=${IMAGE_VERSION} -X ${PKG}/pkg/azurefile.gitCommit=${GIT_COMMIT} -X ${PKG}/pkg/azurefile.buildDate=${BUILD_DATE} -s -w -extldflags '-static'"
GINKGO_FLAGS = -ginkgo.noColor -ginkgo.v
GO111MODULE = on
GOPATH ?= $(shell go env GOPATH)
GOBIN ?= $(GOPATH)/bin
export GOPATH GOBIN

.EXPORT_ALL_VARIABLES:

.PHONY: all
all: azurefile

.PHONY: update
update:
	hack/update-dependencies.sh
	hack/verify-update.sh

.PHONY: verify
verify: 
	hack/verify-all.sh

.PHONY: unit-test
unit-test:
	go test -v -race ./pkg/... ./test/utils/credentials

.PHONY: sanity-test
sanity-test: azurefile
	go test -v -timeout=10m ./test/sanity

.PHONY: integration-test
integration-test: azurefile
	go test -v -timeout=10m ./test/integration

.PHONY: e2e-test
e2e-test:
	go test -v -timeout=30m ./test/e2e ${GINKGO_FLAGS}

.PHONY: e2e-bootstrap
e2e-bootstrap: install-helm
	# Only build and push the image if it does not exist in the registry
	docker pull $(IMAGE_TAG) || make azurefile-container push
	# Timeout after waiting 15 minutes = 900 seconds
	helm install charts/latest/azurefile-csi-driver -n azurefile-csi-driver --namespace kube-system --wait --timeout 900 \
		--set image.azurefile.pullPolicy=IfNotPresent \
		--set image.azurefile.repository=$(REGISTRY)/$(IMAGE_NAME) \
		--set image.azurefile.tag=$(IMAGE_VERSION)

.PHONY: install-helm
install-helm:
	# Use v2.11.0 helm to match tiller's version in clusters made by aks-engine
	curl https://raw.githubusercontent.com/helm/helm/master/scripts/get | DESIRED_VERSION=v2.11.0 bash
	# Make sure tiller is ready
	kubectl wait pod -l name=tiller --namespace kube-system --for condition=ready
	helm version

.PHONY: e2e-teardown
e2e-teardown:
	helm delete --purge azurefile-csi-driver

.PHONY: azurefile
azurefile:
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags ${LDFLAGS} -o _output/azurefileplugin ./pkg/azurefileplugin

.PHONY: azurefile-windows
azurefile-windows:
	CGO_ENABLED=0 GOOS=windows go build -a -ldflags ${LDFLAGS} -o _output/azurefileplugin.exe ./pkg/azurefileplugin

.PHONY: container
container: azurefile
	docker build --no-cache -t $(IMAGE_TAG) -f ./pkg/azurefileplugin/Dockerfile .

.PHONY: azurefile-container
azurefile-container: azurefile
	docker build --no-cache -t $(IMAGE_TAG) -f ./pkg/azurefileplugin/Dockerfile .

.PHONY: azurefile-container-windows
azurefile-container-windows: azurefile-windows
	docker build --no-cache -t $(IMAGE_TAG) -f ./pkg/azurefileplugin/Windows.Dockerfile .

.PHONY: push
push:
	docker push $(IMAGE_TAG)

.PHONY: push-latest
push-latest:
	docker tag $(IMAGE_TAG) $(IMAGE_TAG_LATEST)
	docker push $(IMAGE_TAG_LATEST)

.PHONY: build-push
build-push: azurefile-container
	docker tag $(IMAGE_TAG) $(IMAGE_TAG_LATEST)
	docker push $(IMAGE_TAG_LATEST)

.PHONY: clean
clean:
	go clean -r -x
	-rm -rf _output
