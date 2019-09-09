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

function cleanup {
  echo 'pkill -f azurefileplugin'
  pkill -f azurefileplugin
}

endpoint='tcp://127.0.0.1:10000'
if [[ "$#" -gt 0 ]]; then
  endpoint="$1"
fi

target_path='/tmp/testmount'
volname=$(date +%s)
volname="citest-$volname"
if [[ "$#" -gt 1 ]]; then
  target_path="$2"
fi

cloud='AzurePublicCloud'
if [[ "$#" -gt 2 ]]; then
  cloud="$3"
fi

echo "Begin to run integration test on $cloud..."

# Run CSI driver as a background service
_output/azurefileplugin --endpoint "$endpoint" --nodeid CSINode -v=5 &
trap cleanup EXIT

if [[ "$cloud" == 'AzureChinaCloud' ]]; then
  sleep 25
else
  sleep 5
fi

# Begin to run CSI functions one by one
echo 'Create volume test:'
value=$("$csc" controller new --endpoint "$endpoint" --cap 1,block "$volname" --req-bytes 2147483648 --params skuname=Standard_LRS)
sleep 15

volumeid=$(echo "$value" | awk '{print $1}' | sed 's/"//g')
echo "Got volume id: $volumeid"

"$csc" controller validate-volume-capabilities --endpoint "$endpoint" --cap 1,block "$volumeid"

if [[ "$cloud" != 'AzureChinaCloud' ]]; then
  # azure file mount/unmount on travis VM does not work against AzureChinaCloud
  echo 'Mount volume test:'
  "$csc" node publish --endpoint "$endpoint" --cap 1,block --target-path "$target_path" "$volumeid"
  sleep 2

  echo 'Unmount volume test:'
  "$csc" node unpublish --endpoint "$endpoint" --target-path "$target_path" "$volumeid"
  sleep 2
fi

echo 'Delete volume test:'
"$csc" controller del --endpoint "$endpoint" "$volumeid"
sleep 15

"$csc" identity plugin-info --endpoint "$endpoint"
"$csc" node get-info --endpoint "$endpoint"

echo "Integration test on $cloud is complete."
