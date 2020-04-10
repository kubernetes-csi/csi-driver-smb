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
REGISTRY_NAME = $(shell echo $(REGISTRY) | sed "s/.azurecr.io//g")
IMAGE_NAME = azurefile-csi
IMAGE_VERSION ?= v0.7.0
# Use a custom version for E2E tests if we are testing in CI
ifdef CI
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
DOCKER_CLI_EXPERIMENTAL = enabled
export GOPATH GOBIN GO111MODULE DOCKER_CLI_EXPERIMENTAL

.EXPORT_ALL_VARIABLES:

.PHONY: all
all: azurefile

.PHONY: update
update:
	hack/update-dependencies.sh
	hack/verify-update.sh

.PHONY: verify
verify: unit-test
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
	go test -v -timeout=0 ./test/e2e ${GINKGO_FLAGS}

.PHONY: e2e-bootstrap
e2e-bootstrap: install-helm
	docker pull $(IMAGE_TAG) || make azurefile-container push
ifdef TEST_WINDOWS
	helm install azurefile-csi-driver charts/latest/azurefile-csi-driver --namespace kube-system --wait --timeout=15m -v=5 --debug \
		--set image.azurefile.repository=$(REGISTRY)/$(IMAGE_NAME) \
		--set image.azurefile.tag=$(IMAGE_VERSION) \
		--set windows.enabled=true \
		--set linux.enabled=false \
		--set controller.replicas=1
else
	helm install azurefile-csi-driver charts/latest/azurefile-csi-driver --namespace kube-system --wait --timeout=15m -v=5 --debug \
		--set image.azurefile.repository=$(REGISTRY)/$(IMAGE_NAME) \
		--set image.azurefile.tag=$(IMAGE_VERSION) \
		--set snapshot.enabled=true
endif

.PHONY: install-helm
install-helm:
	curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash

.PHONY: e2e-teardown
e2e-teardown:
	helm delete azurefile-csi-driver --namespace kube-system

.PHONY: azurefile
azurefile:
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags ${LDFLAGS} -o _output/azurefileplugin ./pkg/azurefileplugin

.PHONY: azurefile-windows
azurefile-windows:
	CGO_ENABLED=0 GOOS=windows go build -a -ldflags ${LDFLAGS} -o _output/azurefileplugin.exe ./pkg/azurefileplugin

.PHONY: azurefile-container
azurefile-container:
ifdef CI
	az acr login --name $(REGISTRY_NAME)
	make azurefile azurefile-windows
	az acr build --registry $(REGISTRY_NAME) -t $(IMAGE_TAG)-linux-amd64 -f ./pkg/azurefileplugin/Dockerfile --platform linux .
	az acr build --registry $(REGISTRY_NAME) -t $(IMAGE_TAG)-windows-1809-amd64 -f ./pkg/azurefileplugin/Windows.Dockerfile --platform windows .
	docker manifest create $(IMAGE_TAG) $(IMAGE_TAG)-linux-amd64 $(IMAGE_TAG)-windows-1809-amd64
	docker manifest inspect $(IMAGE_TAG)
else
ifdef TEST_WINDOWS
	make azurefile-windows
	docker build --no-cache -t $(IMAGE_TAG) -f ./pkg/azurefileplugin/Windows.Dockerfile .
else
	make azurefile
	docker build --no-cache -t $(IMAGE_TAG) -f ./pkg/azurefileplugin/Dockerfile .
endif
endif

.PHONY: push
push:
ifdef CI
	docker manifest push --purge $(IMAGE_TAG)
else
	docker push $(IMAGE_TAG)
endif

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
