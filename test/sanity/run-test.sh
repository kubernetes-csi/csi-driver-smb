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

 if [ -v GOPATH ]; then
		mkdir $GOPATH/src/github.com/kubernetes-csi
		pushd $GOPATH/src/github.com/kubernetes-csi
		git clone https://github.com/kubernetes-csi/csi-test.git -b v1.1.0
        pushd $GOPATH/src/github.com/kubernetes-csi/csi-test/cmd/csi-sanity
		make && make install 
		popd
		popd
fi

 endpoint="unix:///tmp/csi.sock"

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
