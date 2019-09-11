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

set -euo pipefail

sudo apt update && sudo apt install cifs-utils procps -y
GO111MODULE=off go get github.com/rexray/gocsi/csc

sudo test/integration/run-test.sh 'tcp://127.0.0.1:10000' '/tmp/testmount1' 'AzurePublicCloud'

# Only test on AzureChinaCloud if the following environment variables are set
if [[ -v tenantId_china ]] && [[ -v subscriptionId_china ]] && [[ -v aadClientId_china ]] && [[ -v aadClientSecret_china ]] && [[ -v resourceGroup_china ]] && [[ -v location_china ]]; then
  sudo test/integration/run-test.sh 'tcp://127.0.0.1:10001' '/tmp/testmount2' 'AzureChinaCloud'
fi
