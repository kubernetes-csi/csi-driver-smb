# Copyright 2018 The Kubernetes Authors.
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

IMAGE_NAME = quay.io/k8scsi/mock-driver
IMAGE_VERSION = canary
APP := ./bin/mock


ifdef V
TESTARGS = -v -args -alsologtostderr -v 5
else
TESTARGS =
endif

all: $(APP)

$(APP):
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o $(APP) ./mock/main.go

clean:
	rm -rf bin

container: $(APP)
	docker build -f Dockerfile.mock -t $(IMAGE_NAME):$(IMAGE_VERSION) .

push: container
	docker push $(IMAGE_NAME):$(IMAGE_VERSION)

test: $(APP)
	files=$$(find ./ -name '*.go' | grep -v '^./vendor' ); \
        if [ $$(gofmt -d $$files | wc -l) -ne 0 ]; then \
                echo "formatting errors:"; \
                gofmt -d $$files; \
                false; \
        fi
	go vet $$(go list ./... | grep -v vendor)
	go test $$(go list ./... | grep -v vendor | grep -v "cmd/csi-sanity")
	./hack/e2e.sh

.PHONY: all clean container push test
