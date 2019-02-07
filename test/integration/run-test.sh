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

# run CSI driver as a background service
_output/azurefileplugin --endpoint $endpoint --nodeid CSINode -v=5 &
sleep 3

# begin to run CSI functions one by one
if [ ! -z $aadClientSecret ]; then
	echo "create volume test:"
	value=`$csc controller new --endpoint $endpoint --cap 1,block CSIVolumeName --req-bytes 2147483648 --params skuname=Standard_LRS`
	retcode=$?
	if [ $retcode -gt 0 ]; then
		exit $retcode
	fi
	sleep 15

	volumeid=`echo $value | awk '{print $1}' | sed 's/"//g'`
	echo "got volume id: $volumeid"

	echo "mount volume test:"
	$csc node publish --endpoint $endpoint --cap 1,block --target-path ~/testmount $volumeid
	retcode=$?
	if [ $retcode -gt 0 ]; then
		exit $retcode
	fi
	sleep 2

	echo "unmount volume test:"
	$csc node unpublish --endpoint $endpoint --target-path ~/testmount $volumeid
	retcode=$?
	if [ $retcode -gt 0 ]; then
		exit $retcode
	fi
	sleep 2

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

$csc controller validate-volume-capabilities --endpoint $endpoint --cap 1,block CSIVolumeID
retcode=$?
if [ $retcode -gt 0 ]; then
	exit $retcode
fi

$csc node get-info --endpoint $endpoint
retcode=$?
if [ $retcode -gt 0 ]; then
	exit $retcode
fi
