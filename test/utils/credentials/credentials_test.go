/*
Copyright 2020 The Kubernetes Authors.

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

package credentials

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

const (
	fakeAzureCredentials = `
	[Creds]
	ClientID = "df7269f2-xxxx-xxxx-xxxx-0f12a7d97404"
	ClientSecret = "8c416dc5-xxxx-xxxx-xxxx-d77069e2a255"
	TenantID = "72f988bf-xxxx-xxxx-xxxx-2d7cd011db47"
	SubscriptionID = "b9d2281e-xxxx-xxxx-xxxx-0d50377cdf76"
	StorageAccountName = "TestStorageAccountName"
	StorageAccountKey = "TestStorageAccountKey"
	`
)

func TestCreateAzureCredentialFileOnAzureChinaCloud(t *testing.T) {
	t.Run("WithAzureCredentials", func(t *testing.T) {
		os.Setenv(tenantIDChinaEnvVar, "")
		os.Setenv(subscriptionIDChinaEnvVar, "")
		os.Setenv(aadClientIDChinaEnvVar, "")
		os.Setenv(aadClientSecretChinaEnvVar, "")
		os.Setenv(resourceGroupChinaEnvVar, "test-resource-group")
		os.Setenv(locationChinaEnvVar, "test-location")
		withAzureCredentials(t, true)
	})

	t.Run("WithEnvironmentVariables", func(t *testing.T) {
		os.Setenv(tenantIDChinaEnvVar, "test-tenant-id")
		os.Setenv(subscriptionIDChinaEnvVar, "test-subscription-id")
		os.Setenv(aadClientIDChinaEnvVar, "test-aad-client-id")
		os.Setenv(aadClientSecretChinaEnvVar, "test-aad-client-secret")
		os.Setenv(resourceGroupChinaEnvVar, "test-resource-group")
		os.Setenv(locationChinaEnvVar, "test-location")
		withEnvironmentVariables(t, true)
	})
}

func TestCreateAzureCredentialFileOnAzurePublicCloud(t *testing.T) {
	t.Run("WithAzureCredentials", func(t *testing.T) {
		os.Setenv(tenantIDEnvVar, "")
		os.Setenv(subscriptionIDEnvVar, "")
		os.Setenv(aadClientIDEnvVar, "")
		os.Setenv(aadClientSecretEnvVar, "")
		os.Setenv(resourceGroupEnvVar, "test-resource-group")
		os.Setenv(locationEnvVar, "test-location")
		withAzureCredentials(t, false)
	})

	t.Run("WithEnvironmentVariables", func(t *testing.T) {
		os.Setenv(tenantIDEnvVar, "test-tenant-id")
		os.Setenv(subscriptionIDEnvVar, "test-subscription-id")
		os.Setenv(aadClientIDEnvVar, "test-aad-client-id")
		os.Setenv(aadClientSecretEnvVar, "test-aad-client-secret")
		os.Setenv(resourceGroupEnvVar, "test-resource-group")
		os.Setenv(locationEnvVar, "test-location")
		withEnvironmentVariables(t, false)
	})
}

func withAzureCredentials(t *testing.T, isAzureChinaCloud bool) {
	tempFile, err := ioutil.TempFile("", "azure.toml")
	assert.NoError(t, err)
	defer func() {
		err := os.Remove(tempFile.Name())
		assert.NoError(t, err)
	}()

	os.Setenv("AZURE_CREDENTIALS", tempFile.Name())

	_, err = tempFile.Write([]byte(fakeAzureCredentials))
	assert.NoError(t, err)

	creds, err := CreateAzureCredentialFile(isAzureChinaCloud)
	assert.NoError(t, err)
	defer func() {
		err := DeleteAzureCredentialFile()
		assert.NoError(t, err)
	}()

	var cloud string
	if isAzureChinaCloud {
		cloud = AzureChinaCloud
	} else {
		cloud = AzurePublicCloud
	}

	assert.Equal(t, cloud, creds.Cloud)
	assert.Equal(t, "72f988bf-xxxx-xxxx-xxxx-2d7cd011db47", creds.TenantID)
	assert.Equal(t, "b9d2281e-xxxx-xxxx-xxxx-0d50377cdf76", creds.SubscriptionID)
	assert.Equal(t, "df7269f2-xxxx-xxxx-xxxx-0f12a7d97404", creds.AADClientID)
	assert.Equal(t, "8c416dc5-xxxx-xxxx-xxxx-d77069e2a255", creds.AADClientSecret)
	assert.Equal(t, "test-resource-group", creds.ResourceGroup)
	assert.Equal(t, "test-location", creds.Location)

	azureCredentialFileContent, err := ioutil.ReadFile(TempAzureCredentialFilePath)
	assert.NoError(t, err)

	const expectedAzureCredentialFileContent = `
	{
		"cloud": "{{.Cloud}}",
		"tenantId": "72f988bf-xxxx-xxxx-xxxx-2d7cd011db47",
		"aadClientId": "df7269f2-xxxx-xxxx-xxxx-0f12a7d97404",
		"subscriptionId": "b9d2281e-xxxx-xxxx-xxxx-0d50377cdf76",
		"aadClientSecret": "8c416dc5-xxxx-xxxx-xxxx-d77069e2a255",
		"resourceGroup": "test-resource-group",
		"location": "test-location"
	}
	`
	tmpl := template.New("expectedAzureCredentialFileContent")
	tmpl, err = tmpl.Parse(expectedAzureCredentialFileContent)
	assert.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, struct {
		Cloud string
	}{
		cloud,
	})
	assert.NoError(t, err)
	assert.JSONEq(t, buf.String(), string(azureCredentialFileContent))
}

func withEnvironmentVariables(t *testing.T, isAzureChinaCloud bool) {
	creds, err := CreateAzureCredentialFile(isAzureChinaCloud)
	defer func() {
		err := DeleteAzureCredentialFile()
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)

	var cloud string
	if isAzureChinaCloud {
		cloud = AzureChinaCloud
	} else {
		cloud = AzurePublicCloud
	}

	assert.Equal(t, cloud, creds.Cloud)
	assert.Equal(t, "test-tenant-id", creds.TenantID)
	assert.Equal(t, "test-subscription-id", creds.SubscriptionID)
	assert.Equal(t, "test-aad-client-id", creds.AADClientID)
	assert.Equal(t, "test-aad-client-secret", creds.AADClientSecret)
	assert.Equal(t, "test-resource-group", creds.ResourceGroup)
	assert.Equal(t, "test-location", creds.Location)

	azureCredentialFileContent, err := ioutil.ReadFile(TempAzureCredentialFilePath)
	assert.NoError(t, err)

	const expectedAzureCredentialFileContent = `
	{
		"cloud": "{{.Cloud}}",
		"tenantId": "test-tenant-id",
		"subscriptionId": "test-subscription-id",
		"aadClientId": "test-aad-client-id",
		"aadClientSecret": "test-aad-client-secret",
		"resourceGroup": "test-resource-group",
		"location": "test-location"
	}
	`
	tmpl := template.New("expectedAzureCredentialFileContent")
	tmpl, err = tmpl.Parse(expectedAzureCredentialFileContent)
	assert.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, struct {
		Cloud string
	}{
		cloud,
	})
	assert.NoError(t, err)
	assert.JSONEq(t, buf.String(), string(azureCredentialFileContent))
}
