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

endpoint="tcp://127.0.0.1:10000"
# run CSI driver as a background service
export set AZURE_CREDENTIAL_FILE=test/e2e/azure.json

sed -i "s/tenantId-input/$tenantId/g" $AZURE_CREDENTIAL_FILE
sed -i "s/subscriptionId-input/$subscriptionId/g" $AZURE_CREDENTIAL_FILE
sed -i "s/aadClientId-input/$aadClientId/g" $AZURE_CREDENTIAL_FILE
sed -i "s/aadClientSecret-input/$aadClientSecret/g" $AZURE_CREDENTIAL_FILE
sed -i "s/resourceGroup-input/$resourceGroup/g" $AZURE_CREDENTIAL_FILE

_output/azurefileplugin --endpoint $endpoint --nodeid CSINode -v=5 &
sleep 3

# begin to run CSI functions one by one
$GOPATH/bin/csc identity plugin-info --endpoint $endpoint
retcode=$?
if [ $retcode -gt 0 ]; then
	exit $retcode
fi

$GOPATH/bin/csc controller validate-volume-capabilities --endpoint $endpoint --cap 1,block CSIVolumeID
retcode=$?
if [ $retcode -gt 0 ]; then
	exit $retcode
fi

$GOPATH/bin/csc node get-info --endpoint $endpoint
retcode=$?
if [ $retcode -gt 0 ]; then
	exit $retcode
fi
