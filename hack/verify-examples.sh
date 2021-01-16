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

echo "begin to create deployment examples ..."
kubectl apply -f deploy/example/storageclass-smb.yaml
kubectl apply -f deploy/example/pvc-smb.yaml
kubectl apply -f deploy/example/deployment.yaml
kubectl apply -f deploy/example/statefulset.yaml
kubectl apply -f deploy/example/statefulset-nonroot.yaml

echo "sleep 90s ..."
sleep 90

kubectl get pods --field-selector status.phase=Running | grep deployment-blob
kubectl get pods --field-selector status.phase=Running | grep statefulset-smb-0
kubectl get pods --field-selector status.phase=Running | grep statefulset-smb-nonroot-0

echo "deployment examples running completed."
