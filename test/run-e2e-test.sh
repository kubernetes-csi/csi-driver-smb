#!/bin/bash

# Copyright 2020 The Kubernetes Authors.
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

set -euo pipefail

PROJECT_ROOT=$(git rev-parse --show-toplevel)
GCE_PROJECT=$(gcloud config get-value project)

configure_docker() {
    gcloud auth configure-docker
}

setup_e2e() {
    # If run in prow, need to use kubernetes_e2e.py to set up the project and kubernetes automatically.
    # If run locally, start a k8s cluster with Windows nodes.
    make -C $PROJECT_ROOT e2e-bootstrap
    make -C $PROJECT_ROOT install-smb-provisioner
    make -C $PROJECT_ROOT create-metrics-svc
}

export TEST_WINDOWS=true
export REGISTRY=gcr.io/$GCE_PROJECT

configure_docker
setup_e2e
make -C $PROJECT_ROOT e2e-test
