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

PKG = github.com/kubernetes-csi/csi-driver-smb
GIT_COMMIT ?= $(shell git rev-parse HEAD)
REGISTRY ?= andyzhangx
REGISTRY_NAME = $(shell echo $(REGISTRY) | sed "s/.azurecr.io//g")
IMAGE_NAME ?= smb-csi
IMAGE_VERSION ?= v0.3.0
# Use a custom version for E2E tests if we are testing in CI
ifdef CI
ifndef PUBLISH
override IMAGE_VERSION := e2e-$(GIT_COMMIT)
endif
endif
IMAGE_TAG = $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_VERSION)
IMAGE_TAG_LATEST = $(REGISTRY)/$(IMAGE_NAME):latest
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS ?= "-X ${PKG}/pkg/smb.driverVersion=${IMAGE_VERSION} -X ${PKG}/pkg/smb.gitCommit=${GIT_COMMIT} -X ${PKG}/pkg/smb.buildDate=${BUILD_DATE} -s -w -extldflags '-static'"
GINKGO_FLAGS = -ginkgo.noColor -ginkgo.v
GO111MODULE = on
GOPATH ?= $(shell go env GOPATH)
GOBIN ?= $(GOPATH)/bin
DOCKER_CLI_EXPERIMENTAL = enabled
export GOPATH GOBIN GO111MODULE DOCKER_CLI_EXPERIMENTAL

.EXPORT_ALL_VARIABLES:

.PHONY: all
all: smb

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
sanity-test: smb
	test/sanity/run-test.sh

.PHONY: integration-test
integration-test: smb
	sudo -E env "PATH=$$PATH" bash test/integration/run-test.sh

.PHONY: e2e-test
e2e-test:
	go test -v -timeout=0 ./test/e2e ${GINKGO_FLAGS}

.PHONY: e2e-bootstrap
e2e-bootstrap: install-helm
	docker pull $(IMAGE_TAG) || make smb-container push
ifdef TEST_WINDOWS
	helm install smb-csi-driver charts/latest/smb-csi-driver --namespace kube-system --wait --timeout=15m -v=5 --debug \
		--set image.smb.repository=$(REGISTRY)/$(IMAGE_NAME) \
		--set image.smb.tag=$(IMAGE_VERSION) \
		--set windows.enabled=true \
		--set linux.enabled=false \
		--set controller.replicas=1
else
	helm install smb-csi-driver charts/latest/smb-csi-driver --namespace kube-system --wait --timeout=15m -v=5 --debug \
		--set image.smb.repository=$(REGISTRY)/$(IMAGE_NAME) \
		--set image.smb.tag=$(IMAGE_VERSION) \
		--set snapshot.enabled=true
endif

.PHONY: install-helm
install-helm:
	curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash

.PHONY: e2e-teardown
e2e-teardown:
	helm delete smb-csi-driver --namespace kube-system

.PHONY: smb
smb:
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags ${LDFLAGS} -o _output/smbplugin ./pkg/smbplugin

.PHONY: smb-windows
smb-windows:
	CGO_ENABLED=0 GOOS=windows go build -a -ldflags ${LDFLAGS} -o _output/smbplugin.exe ./pkg/smbplugin

.PHONY: container	
container: smb	
	docker build --no-cache -t $(IMAGE_TAG) -f ./pkg/smbplugin/dev.Dockerfile .

.PHONY: smb-container
smb-container:
ifdef CI
	az acr login --name $(REGISTRY_NAME)
	make smb smb-windows
	az acr build --registry $(REGISTRY_NAME) -t $(IMAGE_TAG)-linux-amd64 -f ./pkg/smbplugin/Dockerfile --platform linux .
	az acr build --registry $(REGISTRY_NAME) -t $(IMAGE_TAG)-windows-1809-amd64 -f ./pkg/smbplugin/Windows.Dockerfile --platform windows .
	docker manifest create $(IMAGE_TAG) $(IMAGE_TAG)-linux-amd64 $(IMAGE_TAG)-windows-1809-amd64
	docker manifest inspect $(IMAGE_TAG)
ifdef PUBLISH
	docker manifest create $(IMAGE_TAG_LATEST) $(IMAGE_TAG)-linux-amd64 $(IMAGE_TAG)-windows-1809-amd64
	docker manifest inspect $(IMAGE_TAG_LATEST)
endif	
else
ifdef TEST_WINDOWS
	make smb-windows
	docker build --no-cache -t $(IMAGE_TAG) -f ./pkg/smbplugin/Windows.Dockerfile .
else
	make smb
	docker build --no-cache -t $(IMAGE_TAG) -f ./pkg/smbplugin/Dockerfile .
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
ifdef CI
	docker manifest push --purge $(IMAGE_TAG_LATEST)
else
	docker push $(IMAGE_TAG_LATEST)
endif

.PHONY: build-push
build-push: smb-container
	docker tag $(IMAGE_TAG) $(IMAGE_TAG_LATEST)
	docker push $(IMAGE_TAG_LATEST)

.PHONY: clean
clean:
	go clean -r -x
	-rm -rf _output
