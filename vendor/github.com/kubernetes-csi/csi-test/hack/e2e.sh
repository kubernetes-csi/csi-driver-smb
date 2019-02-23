#!/bin/bash

TESTARGS=$@
UDS="/tmp/e2e-csi-sanity.sock"
CSI_ENDPOINTS="$CSI_ENDPOINTS ${UDS}"
CSI_MOCK_VERSION="master"

#
# $1 - endpoint for mock.
# $2 - endpoint for csi-sanity in Grpc format.
#      See https://github.com/grpc/grpc/blob/master/doc/naming.md
runTest()
{
	CSI_ENDPOINT=$1 ./bin/mock &
	local pid=$!

	./cmd/csi-sanity/csi-sanity $TESTARGS --csi.endpoint=$2; ret=$?
	kill -9 $pid

	if [ $ret -ne 0 ] ; then
		exit $ret
	fi
}

runTestWithCreds()
{
	CSI_ENDPOINT=$1 CSI_ENABLE_CREDS=true ./bin/mock &
	local pid=$!

	./cmd/csi-sanity/csi-sanity $TESTARGS --csi.endpoint=$2 --csi.secrets=mock/mocksecret.yaml; ret=$?
	kill -9 $pid

	if [ $ret -ne 0 ] ; then
		exit $ret
	fi
}

runTestAPI()
{
	CSI_ENDPOINT=$1 ./bin/mock &
	local pid=$!

	GOCACHE=off go test -v ./hack/_apitest/api_test.go; ret=$?

	if [ $ret -ne 0 ] ; then
		exit $ret
	fi

	GOCACHE=off go test -v ./hack/_embedded/embedded_test.go; ret=$?
	kill -9 $pid

	if [ $ret -ne 0 ] ; then
		exit $ret
	fi
}

make

cd cmd/csi-sanity
  make clean install || exit 1
cd ../..

runTest "${UDS}" "${UDS}"
rm -f $UDS

runTestWithCreds "${UDS}" "${UDS}"
rm -f $UDS

runTestAPI "${UDS}"
rm -f $UDS

exit 0
