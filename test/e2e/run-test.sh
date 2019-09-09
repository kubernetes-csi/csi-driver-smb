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

kubectl get daemonsets csi-azurefile-node -n kube-system
if [[ "$?" -ne 0 ]]; then
  echo 'AzureFile csi driver daemonset not found'
  echo 'Installing AzureFile csi driver'
  deploy/install-driver.sh
  echo 'AzureFile csi driver installed'
fi

# Fetching ginkgo for running the test
go install -v github.com/onsi/ginkgo/ginkgo
export KUBECONFIG="$HOME/.kube/config"
"$GOBIN"/ginkgo test/e2e
exit "$?"
