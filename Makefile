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

PKG=github.com/kubernetes-sigs/azurefile-csi-driver
REGISTRY_NAME=andyzhangx
IMAGE_NAME=azurefile-csi
IMAGE_VERSION=v0.4.0
IMAGE_TAG=$(REGISTRY_NAME)/$(IMAGE_NAME):$(IMAGE_VERSION)
IMAGE_TAG_LATEST=$(REGISTRY_NAME)/$(IMAGE_NAME):latest
GIT_COMMIT?=$(shell git rev-parse HEAD)
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS?="-X ${PKG}/pkg/azurefile.driverVersion=${IMAGE_VERSION} -X ${PKG}/pkg/azurefile.gitCommit=${GIT_COMMIT} -X ${PKG}/pkg/azurefile.buildDate=${BUILD_DATE} -s -w -extldflags '-static'"
GO111MODULE=on

.EXPORT_ALL_VARIABLES:

.PHONY: all azurefile azurefile-container clean
all: azurefile

unit-test:
	go test -v -race ./pkg/...
sanity-test: azurefile
	test/sanity/run-tests-all-clouds.sh
integration-test: azurefile
	test/integration/run-tests-all-clouds.sh
e2e-test:
	test/e2e/run-test.sh
azurefile:
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags ${LDFLAGS} -o _output/azurefileplugin ./pkg/azurefileplugin
azurefile-windows:
	CGO_ENABLED=0 GOOS=windows go build -a -ldflags ${LDFLAGS} -o _output/azurefileplugin.exe ./pkg/azurefileplugin
azurefile-container: azurefile
	docker build --no-cache -t $(IMAGE_TAG) -f ./pkg/azurefileplugin/Dockerfile .
push: azurefile-container
	docker push $(IMAGE_TAG)
push-latest: azurefile-container
	docker push $(IMAGE_TAG)
	docker tag $(IMAGE_TAG) $(IMAGE_TAG_LATEST)
	docker push $(IMAGE_TAG_LATEST)
clean:
	go clean -r -x
	-rm -rf _output
update:
	hack/update-dependencies.sh
verify: update
	hack/verify-all.sh
