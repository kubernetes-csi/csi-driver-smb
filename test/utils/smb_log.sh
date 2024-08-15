#!/bin/bash

# Copyright 2020 The Kubernetes Authors.
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

cleanup() {
    echo "hit unexpected error during log print, exit 0"
    exit 0
}

trap cleanup ERR

NS=kube-system
CONTAINER=smb
DRIVER=smb
if [[ "$#" -gt 0 ]]; then
    DRIVER=$1
fi

echo "print out all nodes status ..."
kubectl get nodes -o wide
echo "======================================================================================"

echo "print out all default namespace pods status ..."
kubectl get pods -n default -o wide
echo "======================================================================================"

echo "print out all $NS namespace pods status ..."
kubectl get pods -n${NS} -o wide
echo "======================================================================================"

echo "print out csi-$DRIVER-controller pods ..."
echo "======================================================================================"
LABEL="app=csi-$DRIVER-controller"
kubectl get pods -n${NS} -l${LABEL} \
    | awk 'NR>1 {print $1}' \
    | xargs -I {} kubectl logs {} --prefix -c${CONTAINER} -n${NS}

echo "print out csi-$DRIVER-node logs ..."
echo "======================================================================================"
LABEL="app=csi-$DRIVER-node"
kubectl get pods -n${NS} -l${LABEL} \
    | awk 'NR>1 {print $1}' \
    | xargs -I {} kubectl logs {} --prefix -c${CONTAINER} -n${NS}

echo "print out csi-$DRIVER-node-win events ..."
echo "======================================================================================"
LABEL="app=csi-$DRIVER-node-win"
kubectl describe pods -n${NS} -l${LABEL}

echo "print out csi-$DRIVER-node-win logs ..."
echo "======================================================================================"
LABEL="app=csi-$DRIVER-node-win"
kubectl get pods -n${NS} -l${LABEL} \
    | awk 'NR>1 {print $1}' \
    | xargs -I {} kubectl logs {} --prefix -c${CONTAINER} -n${NS}

echo "print out service logs ..."
echo "======================================================================================"
kubectl get service -A

echo "print out metrics ..."
echo "======================================================================================"
ip=`kubectl get svc csi-$DRIVER-controller -n kube-system --no-headers | awk '{print $4}'`
curl http://$ip:29644/metrics
