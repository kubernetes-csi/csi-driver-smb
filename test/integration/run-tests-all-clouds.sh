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

set -eo pipefail

sudo apt update && sudo apt install cifs-utils procps -y
go install -v github.com/rexray/gocsi/csc
export csc="$GOBIN/csc"
export AZURE_CREDENTIAL_FILE='/tmp/azure.json'

if [[ -n "$tenantId" ]] && [[ -n "$subscriptionId" ]] && [[ -n "$aadClientId" ]] && [[ -n "$aadClientSecret" ]] && [[ -n "$resourceGroup" ]] && [[ -n "$location" ]]; then
  hack/create-azure-credential-file.sh 'AzurePublicCloud'
  sudo test/integration/run-test.sh 'tcp://127.0.0.1:10000' '/tmp/testmount1' 'AzurePublicCloud'
fi

if [[ -n "$tenantId_china" ]] && [[ -n "$subscriptionId_china" ]] && [[ -n "$aadClientId_china" ]] && [[ -n "$aadClientSecret_china" ]] && [[ -n "$resourceGroup_china" ]] && [[ -n "$location_china" ]]; then
  hack/create-azure-credential-file.sh 'AzureChinaCloud'
  sudo test/integration/run-test.sh 'tcp://127.0.0.1:10001' '/tmp/testmount2' 'AzureChinaCloud'
fi
