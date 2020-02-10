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

apt update && apt install yamllint -y

yamllint -f parsable deploy/*.yaml | grep -v "line too long" > /tmp/yamllint.log
cat /tmp/yamllint.log
linecount=`cat /tmp/yamllint.log | grep -v "line too long" | wc -l`
if [ $linecount -gt 0 ]; then
	echo "yaml files under deploy/ are not linted"
	exit 1
fi

yamllint -f parsable charts/latest/azurefile-csi-driver/templates/*.yaml | grep -v "line too long" | grep -v "too many spaces inside braces" | grep -v "missing document start" | grep -v "syntax error" > /tmp/yamllint.log
linecount=`cat /tmp/yamllint.log | wc -l`
if [ $linecount -gt 0 ]; then
	echo "yaml files under charts/latest/azuredisk-csi-driver/templates/ are not linted"
	exit 1
fi

echo "Congratulations! All Yaml files have been linted."
