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

GO111MODULE=off go get github.com/rexray/gocsi/csc

cloud='AzurePublicCloud'
if [[ "$#" -gt 0 ]]; then
  cloud="$1"
fi

apt update && apt install cifs-utils procps -y
# test azure file
test/integration/run-test.sh 'tcp://127.0.0.1:10000' "/tmp/stagingtargetpath" "/tmp/targetpath" "skuname=Standard_LRS" "$cloud"

# test vhd disk feature
test/integration/run-test.sh 'tcp://127.0.0.1:10000' "/tmp/stagingtargetpath" "/tmp/targetpath" "skuName=Premium_LRS,fsType=xfs" "$cloud"
