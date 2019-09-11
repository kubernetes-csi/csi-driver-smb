/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sanity

import (
	"testing"

	"github.com/kubernetes-sigs/azurefile-csi-driver/test/azure"
	"github.com/kubernetes-sigs/azurefile-csi-driver/test/credentials"
	"github.com/stretchr/testify/assert"
)

const (
	cloudEnvironment = "AzurePublicCloud"
)

func TestSanity(t *testing.T) {
	c, err := credentials.Get()
	assert.NoError(t, err)

	azureClient, err := azure.GetAzureClient(cloudEnvironment, c.SubscriptionID, c.AADClientID, c.TenantID, c.AADClientSecret)

}
