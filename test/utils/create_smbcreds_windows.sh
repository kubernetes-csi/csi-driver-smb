#!/bin/bash

# Copyright 2025 The Kubernetes Authors.
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

set -e
username=`echo YW5keXNzZGZpbGUK | base64 -d`
pwd=`echo RDVyY1pFMkZ1UlRZVktmaTd4SlZCb1VwdUhLZkRpQUhxZmZzaEVEMXlrQXNPMktaKzZvS25nemF5alZpL1hhSU5zaWVtUGlHSUp5ZkhGcTZUSm5rOUE9PQo= | base64 -d`
kubectl delete secret smbcreds --ignore-not-found -n default
kubectl create secret generic smbcreds --from-literal username=$username --from-literal password=$pwd --from-literal mountOptions="dir_mode=0777,file_mode=0777,uid=0,gid=0,mfsymlinks" -n default

