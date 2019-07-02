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

csc=$GOPATH/bin/csc

endpoint="tcp://127.0.0.1:10000"
if [ $# -gt 0 ]; then
	endpoint=$1
fi

target_path="/tmp/testmount"
volname=`date +%s`
volname="citest-$volname"
if [ $# -gt 1 ]; then
	target_path=$2
fi

cloud="AzurePublicCloud"
if [ $# -gt 2 ]; then
        cloud=$3
fi

echo "being to run integration test on $cloud ..."
# run CSI driver as a background service
_output/azurefileplugin --endpoint $endpoint --nodeid CSINode -v=5 &
if [ $cloud = "AzureChinaCloud" ]; then
	sleep 25
else
	sleep 5
fi

# begin to run CSI functions one by one
if [ -v aadClientSecret ]; then
	echo "create volume test:"
	value=`$csc controller new --endpoint $endpoint --cap 1,block $volname --req-bytes 2147483648 --params skuname=Standard_LRS`
	retcode=$?
	if [ $retcode -gt 0 ]; then
		exit $retcode
	fi
	sleep 15

	volumeid=`echo $value | awk '{print $1}' | sed 's/"//g'`
	echo "got volume id: $volumeid"

	$csc controller validate-volume-capabilities --endpoint $endpoint --cap 1,block $volumeid
	retcode=$?
	if [ $retcode -gt 0 ]; then
		exit $retcode
	fi

	if [ "$cloud" != "AzureChinaCloud" ]; then
		# azure file mount/unmount on travis VM does not work against AzureChinaCloud
		echo "mount volume test:"
		$csc node publish --endpoint $endpoint --cap 1,block --target-path $target_path $volumeid
		retcode=$?
		if [ $retcode -gt 0 ]; then
			exit $retcode
		fi
		sleep 2

		echo "unmount volume test:"
		$csc node unpublish --endpoint $endpoint --target-path $target_path $volumeid
		retcode=$?
		if [ $retcode -gt 0 ]; then
			exit $retcode
		fi
		sleep 2
	fi

	echo "delete volume test:"
	$csc controller del --endpoint $endpoint $volumeid
	retcode=$?
	if [ $retcode -gt 0 ]; then
		exit $retcode
	fi
	sleep 15
fi

$csc identity plugin-info --endpoint $endpoint
retcode=$?
if [ $retcode -gt 0 ]; then
	exit $retcode
fi

$csc node get-info --endpoint $endpoint
retcode=$?
if [ $retcode -gt 0 ]; then
	exit $retcode
fi

# kill azurefileplugin first
echo "pkill -f azurefileplugin"
/usr/bin/pkill -f azurefileplugin

echo "integration test on $cloud is completed."
