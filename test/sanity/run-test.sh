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
  rm -rf csi-test
}

readonly endpoint='unix:///tmp/csi.sock'
nodeid='CSINode'
if [[ "$#" -gt 0 ]]; then
  nodeid="$1"
fi

_output/azurefileplugin --endpoint "$endpoint" --nodeid "$nodeid" -v=5 &
trap cleanup EXIT

<<<<<<< HEAD
<<<<<<< HEAD
node="CSINode"
if [ $# -gt 0 ]; then
	node=$1
fi

 echo "begin to run sanity test ..."

 sudo _output/azurefileplugin --endpoint $endpoint --nodeid $node -v=5 &

 sudo $GOPATH/src/github.com/kubernetes-csi/csi-test/cmd/csi-sanity/csi-sanity --ginkgo.v --csi.endpoint=$endpoint

 retcode=$?

 if [ $retcode -ne 0 ]; then
	exit $retcode
fi

 # kill azurefileplugin first
echo "pkill -f azurefileplugin"
sudo /usr/bin/pkill -f azurefileplugin

echo "sanity test is completed."
=======
# Skip "should fail when requesting to create a snapshot with already existing name and different SourceVolumeId.", because azurefile cannot specify the snapshot name.
echo "Begin to run sanity test..."
<<<<<<< HEAD
sudo csi-sanity --ginkgo.v --csi.endpoint="$endpoint" -ginkgo.skip='should fail when requesting to create a snapshot with already existing name and different SourceVolumeId.'
>>>>>>> 84c0eaef... Modify scripts for sanity test
=======
=======
echo 'Begin to run sanity test...'
# Skip "should fail when requesting to create a snapshot with already existing name and different SourceVolumeId.", because azurefile cannot specify the snapshot name.
<<<<<<< HEAD
>>>>>>> 4ab3645c... Clean up scripts for sanity and integration test
sudo csi-test/cmd/csi-sanity/csi-sanity --ginkgo.v --csi.endpoint="$endpoint" -ginkgo.skip='should fail when requesting to create a snapshot with already existing name and different SourceVolumeId.'
>>>>>>> f84d9226... Cache go modules and binaries in Travis CI
=======
"$CSI_SANITY_BIN" --ginkgo.v --csi.endpoint="$endpoint" -ginkgo.skip='should fail when requesting to create a snapshot with already existing name and different SourceVolumeId.'
>>>>>>> 8e7de4b0... Minor clean up
