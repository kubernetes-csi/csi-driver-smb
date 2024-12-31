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

ver="master"
if [[ "$#" -gt 0 ]]; then
  ver="$1"
fi

repo="https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/$ver/deploy"

windowsMode="csi-proxy"
if [[ "$#" -gt 1 ]]; then
  if [[ "$2" == *"local"* ]]; then
    echo "use local deploy"
    repo="./deploy"
  fi
  if [[ "$2" == *"hostprocess"* ]]; then
    windowsMode="hostprocess"
  fi
fi

if [ $ver != "master" ]; then
  repo="$repo/$ver"
fi

echo "Installing SMB CSI driver, version: $ver ..."
kubectl apply -f $repo/rbac-csi-smb.yaml
kubectl apply -f $repo/csi-smb-driver.yaml
kubectl apply -f $repo/csi-smb-controller.yaml
kubectl apply -f $repo/csi-smb-node.yaml
if [[ "$windowsMode" == *"hostprocess"* ]]; then
  echo "deploy windows driver with hostprocess mode..."
  kubectl apply -f $repo/csi-smb-node-windows-hostprocess.yaml
else
  echo "deploy windows driver with csi-proxy mode ..."
  kubectl apply -f $repo/csi-smb-node-windows.yaml
fi
echo 'SMB CSI driver installed successfully.'
