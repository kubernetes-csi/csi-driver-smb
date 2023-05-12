#!/bin/bash

# Copyright 2021 The Kubernetes Authors.
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

set -xe

PROJECT_ROOT=$(git rev-parse --show-toplevel)
DRIVER="test"

install_ginkgo () {
    go install github.com/onsi/ginkgo/ginkgo@v1.14.0
}

setup_e2e_binaries() {
    # download k8s external e2e binary
    curl -sL https://dl.k8s.io/release/v1.24.0/kubernetes-test-linux-amd64.tar.gz --output e2e-tests.tar.gz
    tar -xvf e2e-tests.tar.gz && rm e2e-tests.tar.gz

    # test on alternative driver name
    export EXTRA_HELM_OPTIONS=" --set driver.name=$DRIVER.csi.k8s.io --set controller.name=csi-$DRIVER-controller --set linux.dsName=csi-$DRIVER-node --set windows.dsName=csi-$DRIVER-node-win"
    sed -i "s/smb.csi.k8s.io/$DRIVER.csi.k8s.io/g" deploy/example/storageclass-smb.yaml
    sed -i "s/gid=/uid=/g" deploy/example/storageclass-smb.yaml
    make install-smb-provisioner
    make e2e-bootstrap
    sed -i "s/csi-smb-controller/csi-$DRIVER-controller/g" deploy/example/metrics/csi-smb-controller-svc.yaml
    make create-metrics-svc
}

print_logs() {
    bash ./hack/verify-examples.sh
    echo "print out driver logs ..."
    bash ./test/utils/smb_log.sh $DRIVER
}

install_ginkgo
setup_e2e_binaries
trap print_logs EXIT

mkdir -p /tmp/csi
cp deploy/example/storageclass-smb.yaml /tmp/csi/storageclass.yaml
ginkgo -p --progress --v -focus='External.Storage' \
       -skip='\[Disruptive\]||should provision storage with any volume data source|should mount multiple PV pointing to the same storage on the same node' kubernetes/test/bin/e2e.test  -- \
       -storage.testdriver=$PROJECT_ROOT/test/external-e2e/testdriver.yaml \
       --kubeconfig=$KUBECONFIG
