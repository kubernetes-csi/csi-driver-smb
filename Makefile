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

REGISTRY_NAME=andyzhangx
IMAGE_NAME=azurefile-csi
IMAGE_VERSION=v0.1.2-alpha
IMAGE_TAG=$(REGISTRY_NAME)/$(IMAGE_NAME):$(IMAGE_VERSION)
REV=$(shell git describe --long --tags --dirty)

.PHONY: all azurefile azurefile-container clean

all: azurefile

test:
	go test github.com/andyzhangx/azurefile-csi-driver/pkg/... -cover
	go vet github.com/andyzhangx/azurefile-csi-driver/pkg/...
azurefile:
	if [ ! -d ./vendor ]; then dep ensure -vendor-only; fi
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-X github.com/andyzhangx/azurefile-csi-driver/pkg/azurefile.vendorVersion=$(IMAGE_VERSION) -extldflags "-static"' -o _output/azurefileplugin ./pkg/azurefileplugin
azurefile-container: azurefile
	docker build --no-cache -t $(IMAGE_TAG) -f ./pkg/azurefileplugin/Dockerfile .
push: azurefile-container
	docker push $(IMAGE_TAG)
clean:
	go clean -r -x
	-rm -rf _output
