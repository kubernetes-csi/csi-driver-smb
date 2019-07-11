#!/bin/bash

# Copyright 2019 The Kubernetes Authors.
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

set -uo pipefail

kubectl get daemonsets csi-azurefile -n kube-system; checkAzureFileDriver=$?;
if [ $checkAzureFileDriver -ne 0 ]; then
    echo "AzureFile csi driver daemonset not found";
    echo "Installing AzureFile csi driver";
    kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/crd-csi-driver-registry.yaml
    kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/crd-csi-node-info.yaml
    kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/rbac-csi-azurefile-controller.yaml
    kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/csi-azurefile-controller.yaml
    kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/csi-azurefile-node.yaml
    echo "AzureFile csi driver installed";    
fi

# Fetching ginkgo for running the test
GO111MODULE=off go get -u github.com/onsi/ginkgo/ginkgo
# Exporting KUBECONFIG path
export KUBECONFIG=$HOME/.kube/config
# Running the e2e test
ginkgo test/e2e
# Checking for test status
TEST_PASS=$?
if [[ $TEST_PASS -ne 0 ]]; then
    exit 1
fi
