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

if [[ -z "$AZURE_CREDENTIAL_FILE" ]]; then
  export AZURE_CREDENTIAL_FILE='/tmp/azure.json'
  hack/create-azure-credential-file.sh 'AzurePublicCloud'
fi

test/sanity/run-test.sh "$nodeid"
