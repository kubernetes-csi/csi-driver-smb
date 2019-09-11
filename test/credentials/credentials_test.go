package credentials

import (
	"io/ioutil"
	"os"
	"testing"

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

func TestGetWithAzureCredentials(t *testing.T) {
	defer func() {
		err := os.Remove(TempAzureCredentialFilePath)
		assert.NoError(t, err)
	}()

	os.Setenv("tenantId", "")
	os.Setenv("subscriptionId", "")
	os.Setenv("aadClientId", "")
	os.Setenv("aadClientSecret", "")
	os.Setenv("resourceGroup", "test-resource-group")
	os.Setenv("location", "test-location")

	tempFile, err := ioutil.TempFile("", "azure.toml")
	assert.NoError(t, err)
	os.Setenv("AZURE_CREDENTIALS", tempFile.Name())
	defer os.Remove(tempFile.Name())

	_, err = tempFile.Write([]byte(fakeAzureCredentials))
	assert.NoError(t, err)

	creds, err := Get(false)
	assert.NoError(t, err)
	assert.Equal(t, AzurePublicCloud, creds.Cloud)
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
		"cloud": "AzurePublicCloud",
	    "tenantId": "72f988bf-xxxx-xxxx-xxxx-2d7cd011db47",
	    "subscriptionId": "b9d2281e-xxxx-xxxx-xxxx-0d50377cdf76",
	    "aadClientId": "df7269f2-xxxx-xxxx-xxxx-0f12a7d97404",
	    "aadClientSecret": "8c416dc5-xxxx-xxxx-xxxx-d77069e2a255",
	    "resourceGroup": "test-resource-group",
	    "location": "test-location"
	}
	`
	assert.JSONEq(t, expectedAzureCredentialFileContent, string(azureCredentialFileContent))
}

func TestGetWithEnvironmentVariables(t *testing.T) {
	defer func() {
		err := os.Remove(TempAzureCredentialFilePath)
		assert.NoError(t, err)
	}()

	os.Setenv("tenantId", "test-tenant-id")
	os.Setenv("subscriptionId", "test-subscription-id")
	os.Setenv("aadClientId", "test-aad-client-id")
	os.Setenv("aadClientSecret", "test-aad-client-secret")
	os.Setenv("resourceGroup", "test-resource-group")
	os.Setenv("location", "test-location")

	creds, err := Get(false)
	assert.NoError(t, err)
	assert.Equal(t, AzurePublicCloud, creds.Cloud)
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
		"cloud": "AzurePublicCloud",
	    "tenantId": "test-tenant-id",
	    "subscriptionId": "test-subscription-id",
	    "aadClientId": "test-aad-client-id",
	    "aadClientSecret": "test-aad-client-secret",
	    "resourceGroup": "test-resource-group",
	    "location": "test-location"
	}
	`
	assert.JSONEq(t, expectedAzureCredentialFileContent, string(azureCredentialFileContent))
}
