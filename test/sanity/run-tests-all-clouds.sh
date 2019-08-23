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

if [ ! -v AZURE_CREDENTIAL_FILE ]; then
	export set AZURE_CREDENTIAL_FILE=/tmp/azure.json
fi

GO_BIN_PATH=`which go`

# run test on AzurePublicCloud
if [ -v aadClientSecret ]; then
	cp test/integration/azure.json $AZURE_CREDENTIAL_FILE

	sed -i "s/tenantId-input/$tenantId/g" $AZURE_CREDENTIAL_FILE
	sed -i "s/subscriptionId-input/$subscriptionId/g" $AZURE_CREDENTIAL_FILE
	sed -i "s/aadClientId-input/$aadClientId/g" $AZURE_CREDENTIAL_FILE
	sed -i "s#aadClientSecret-input#$aadClientSecret#g" $AZURE_CREDENTIAL_FILE
	sed -i "s/resourceGroup-input/$resourceGroup/g" $AZURE_CREDENTIAL_FILE
	sed -i "s/location-input/$location/g" $AZURE_CREDENTIAL_FILE

	test/sanity/run-test.sh $nodeid
else
	if [ -v subscriptionId ]; then
		echo "skip sanity test in CI env"
	else
		# run test in user mode
		go test -v ./test/sanity/...
	fi
fi
