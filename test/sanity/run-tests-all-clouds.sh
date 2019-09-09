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

if [[ -z "$AZURE_CREDENTIAL_FILE" ]]; then
  export AZURE_CREDENTIAL_FILE=/tmp/azure.json
  cp test/integration/azure.json $AZURE_CREDENTIAL_FILE
  # Run test on AzurePublicCloud
  if [[ ! -z "$tenantId" ]] && [[ ! -z "$subscriptionId" ]] && [[ ! -z "$aadClientId" ]] && [[ ! -z "$aadClientSecret" ]] && [[ ! -z "$resourceGroup" ]] && [[ ! -z "$location" ]]; then
    sed -i "s/tenantId-input/$tenantId/g" $AZURE_CREDENTIAL_FILE
    sed -i "s/subscriptionId-input/$subscriptionId/g" $AZURE_CREDENTIAL_FILE
    sed -i "s/aadClientId-input/$aadClientId/g" $AZURE_CREDENTIAL_FILE
    sed -i "s#aadClientSecret-input#$aadClientSecret#g" $AZURE_CREDENTIAL_FILE
    sed -i "s/resourceGroup-input/$resourceGroup/g" $AZURE_CREDENTIAL_FILE
    sed -i "s/location-input/$location/g" $AZURE_CREDENTIAL_FILE
  else
    echo 'Since $AZURE_CREDENTIAL_FILE is not supplied, $tenantId, $subscriptionId, $aadClientId, $aadClientSecret, $resourceGroup, $location are required to run the sanity test.'
    exit 1
  fi
fi

test/sanity/run-test.sh "$nodeid"
