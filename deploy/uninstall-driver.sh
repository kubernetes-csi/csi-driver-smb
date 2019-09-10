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

echo 'Uninstalling Azure File CSI driver...'
kubectl delete -f deploy/crd-csi-driver-registry.yaml --ignore-not-found
kubectl delete -f deploy/crd-csi-node-info.yaml --ignore-not-found
kubectl delete -f deploy/rbac-csi-azurefile-controller.yaml --ignore-not-found
kubectl delete -f deploy/csi-azurefile-controller.yaml --ignore-not-found
kubectl delete -f deploy/csi-azurefile-node.yaml --ignore-not-found
echo 'Azure File CSI driver uninstalled'
