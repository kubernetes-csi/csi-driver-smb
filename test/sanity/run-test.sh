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

function cleanup {
  echo 'pkill -f smbplugin'
  if [ -z "$GITHUB_ACTIONS" ]
  then
    # if not running on github actions, do not use sudo
    pkill -f smbplugin
  else
    # if running on github actions, use sudo
    sudo pkill -f smbplugin
  fi
  echo 'Deleting CSI sanity test binary'
  rm -rf csi-test
  echo 'Uninstalling samba server on localhost'
  docker rm samba -f
}

trap cleanup EXIT

function install_csi_sanity_bin {
  echo 'Installing CSI sanity test binary...'
  mkdir -p $GOPATH/src/github.com/kubernetes-csi
  pushd $GOPATH/src/github.com/kubernetes-csi
  export GO111MODULE=off
  git clone https://github.com/kubernetes-csi/csi-test.git -b v5.3.1
  pushd csi-test/cmd/csi-sanity
  make install
  popd
  popd
}

function provision_samba_server {
  echo 'Running samba server on localhost'
  docker run -e PERMISSIONS=0777 -p 445:445 --name samba -d andyzhangx/samba:win-fix -s "share;/smbshare/;yes;no;no;all;none" -u "sanity;sanitytestpassword" -p
}

provision_samba_server

if [[ -z "$(command -v csi-sanity)" ]]; then
	install_csi_sanity_bin
fi

readonly endpoint='unix:///tmp/csi.sock'
nodeid='CSINode'
if [[ "$#" -gt 0 ]] && [[ -n "$1" ]]; then
  nodeid="$1"
fi

ARCH=$(uname -p)
if [[ "${ARCH}" == "x86_64" || ${ARCH} == "unknown" ]]; then
  ARCH="amd64"
fi

if [ -z "$GITHUB_ACTIONS" ]
then
  # if not running on github actions, do not use sudo
  _output/${ARCH}/smbplugin --endpoint "$endpoint" --nodeid "$nodeid" -v=5 &
else
  # if running on github actions, use sudo
  sudo _output/${ARCH}/smbplugin --endpoint "$endpoint" --nodeid "$nodeid" -v=5 &
fi

# sleep a while waiting for azurefileplugin start up
sleep 1

echo 'Begin to run sanity test...'
CSI_SANITY_BIN=$GOPATH/bin/csi-sanity
skipTests='create a volume with already existing name and different capacity|should fail when requesting to create a volume with already existing name and different capacity|should fail when the requested volume does not exist'
if [ -z "$GITHUB_ACTIONS" ]
then
  # if not running on github actions, do not use sudo
  "$CSI_SANITY_BIN" --ginkgo.v --csi.secrets="$(pwd)/test/sanity/secrets.yaml" --csi.testvolumeparameters="$(pwd)/test/sanity/params.yaml" --csi.endpoint="$endpoint" --ginkgo.skip="$skipTests"
else
  # if running on github actions, use sudo
  sudo "$CSI_SANITY_BIN" --ginkgo.v --csi.secrets="$(pwd)/test/sanity/secrets.yaml" --csi.testvolumeparameters="$(pwd)/test/sanity/params.yaml" --csi.endpoint="$endpoint" --ginkgo.skip="$skipTests"
fi
