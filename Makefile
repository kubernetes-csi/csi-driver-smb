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
IMAGE_VERSION ?= v0.5.0
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

# Generate all combination of all OS, ARCH, and OSVERSIONS for iteration
ALL_OS = linux windows
ALL_ARCH.linux = amd64 arm64
ALL_OS_ARCH.linux = $(foreach arch, ${ALL_ARCH.linux}, linux-$(arch))
ALL_ARCH.windows = amd64
ALL_OSVERSIONS.windows := 1809 1903 1909 2004
ALL_OS_ARCH.windows = $(foreach arch, $(ALL_ARCH.windows), $(foreach osversion, ${ALL_OSVERSIONS.windows}, windows-${osversion}-${arch}))
ALL_OS_ARCH = $(foreach os, $(ALL_OS), ${ALL_OS_ARCH.${os}})

# The current context of image building
# The architecture of the image
ARCH ?= amd64
# OS Version for the Windows images: 1809, 1903, 1909, 2004
OSVERSION ?= 1809
# Output type of docker buildx build
OUTPUT_TYPE ?= registry

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

.PHONY: deploy-kind
deploy-kind:
	test/utils/deploy-kind.sh

.PHONY: e2e-test
e2e-test:
	go test -v -timeout=0 ./test/e2e ${GINKGO_FLAGS}

.PHONY: e2e-bootstrap
e2e-bootstrap: install-helm
	docker pull $(IMAGE_TAG) || make container-all push-manifest
ifdef TEST_WINDOWS
	helm install csi-driver-smb charts/latest/csi-driver-smb --namespace kube-system --wait --timeout=15m -v=5 --debug \
		--set image.smb.repository=$(REGISTRY)/$(IMAGE_NAME) \
		--set image.smb.tag=$(IMAGE_VERSION) \
		--set windows.enabled=true \
		--set linux.enabled=false \
		--set controller.replicas=1
else
	helm install csi-driver-smb charts/latest/csi-driver-smb --namespace kube-system --wait --timeout=15m -v=5 --debug \
		--set image.smb.repository=$(REGISTRY)/$(IMAGE_NAME) \
		--set image.smb.tag=$(IMAGE_VERSION)
endif

.PHONY: install-helm
install-helm:
	curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash

.PHONY: e2e-teardown
e2e-teardown:
	helm delete csi-driver-smb --namespace kube-system

.PHONY: smb
smb:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) go build -a -ldflags ${LDFLAGS} -o _output/smbplugin ./pkg/smbplugin

.PHONY: smb-windows
smb-windows:
	CGO_ENABLED=0 GOOS=windows go build -a -ldflags ${LDFLAGS} -o _output/smbplugin.exe ./pkg/smbplugin

.PHONY: smb-darwin
smb-darwin:
	CGO_ENABLED=0 GOOS=darwin go build -a -ldflags ${LDFLAGS} -o _output/smbplugin ./pkg/smbplugin

.PHONY: container
container: smb
	docker build --no-cache -t $(IMAGE_TAG) -f ./pkg/smbplugin/dev.Dockerfile .

.PHONY: container-linux
container-linux:
	docker buildx build --pull --output=type=$(OUTPUT_TYPE) --platform="linux/$(ARCH)" \
		-t $(IMAGE_TAG)-linux-$(ARCH) --build-arg ARCH=$(ARCH) -f ./pkg/smbplugin/Dockerfile .

.PHONY: container-windows
container-windows:
	docker buildx build --pull --output=type=$(OUTPUT_TYPE) --platform="windows/$(ARCH)" \
		 -t $(IMAGE_TAG)-windows-$(OSVERSION)-$(ARCH) --build-arg OSVERSION=$(OSVERSION) -f ./pkg/smbplugin/Windows.Dockerfile .

.PHONY: container-all
container-all: smb-windows
	docker buildx rm container-builder || true
	docker buildx create --use --name=container-builder
	for osversion in $(ALL_OSVERSIONS.windows); do \
		OSVERSION=$${osversion} $(MAKE) container-windows; \
	done

	# enable qemu for arm64 build
	docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
	for arch in $(ALL_ARCH.linux); do \
		ARCH=$${arch} $(MAKE) smb; \
		ARCH=$${arch} $(MAKE) container-linux; \
	done
	

.PHONY: push-manifest
push-manifest:
	docker manifest create --amend $(IMAGE_TAG) $(foreach osarch, $(ALL_OS_ARCH), $(IMAGE_TAG)-${osarch})
	# add "os.version" field to windows images (based on https://github.com/kubernetes/kubernetes/blob/master/build/pause/Makefile)
	set -x; \
	registry_prefix=$(shell (echo ${REGISTRY} | grep -Eq ".*\/.*") && echo "docker.io/" || echo ""); \
	manifest_image_folder=`echo "$${registry_prefix}${IMAGE_TAG}" | sed "s|/|_|g" | sed "s/:/-/"`; \
	for arch in $(ALL_ARCH.windows); do \
		for osversion in $(ALL_OSVERSIONS.windows); do \
			BASEIMAGE=mcr.microsoft.com/windows/nanoserver:$${osversion}; \
			full_version=`docker manifest inspect $${BASEIMAGE} | jq -r '.manifests[0].platform["os.version"]'`; \
			sed -i -r "s/(\"os\"\:\"windows\")/\0,\"os.version\":\"$${full_version}\"/" "${HOME}/.docker/manifests/$${manifest_image_folder}/$${manifest_image_folder}-windows-$${osversion}-$${arch}"; \
		done; \
	done
	docker manifest push --purge $(IMAGE_TAG)
ifdef PUBLISH
	docker manifest create $(IMAGE_TAG_LATEST) $(foreach osarch, $(ALL_OS_ARCH), $(IMAGE_TAG)-${osarch})
	docker manifest inspect $(IMAGE_TAG_LATEST)
endif

.PHONY: push-latest
push-latest:
ifdef CI
	docker manifest push --purge $(IMAGE_TAG_LATEST)
else
	docker push $(IMAGE_TAG_LATEST)
endif

.PHONY: clean
clean:
	go clean -r -x
	-rm -rf _output

.PHONY: install-smb-provisioner
install-smb-provisioner:
	kubectl create secret generic smbcreds --from-literal username=USERNAME --from-literal password="PASSWORD"
ifdef TEST_WINDOWS
	kubectl create -f deploy/example/smb-provisioner/smb-server-lb.yaml
else
	kubectl create -f deploy/example/smb-provisioner/smb-server.yaml
endif

.PHONY: create-metrics-svc
create-metrics-svc:
	kubectl create -f deploy/example/metrics/csi-smb-controller-svc.yaml

.PHONY: create-example-deployment
create-example-deployment:
	kubectl apply -f deploy/example/storageclass-smb.yaml
	kubectl apply -f deploy/example/pvc-smb.yaml
	kubectl apply -f deploy/example/deployment.yaml
	kubectl apply -f deploy/example/statefulset.yaml
