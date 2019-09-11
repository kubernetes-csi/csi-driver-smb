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
	expectAzureCredentialFileContent = `
	{
	    "tenantId": "72f988bf-xxxx-xxxx-xxxx-2d7cd011db47",
	    "subscriptionId": "b9d2281e-xxxx-xxxx-xxxx-0d50377cdf76",
	    "aadClientId": "df7269f2-xxxx-xxxx-xxxx-0f12a7d97404",
	    "aadClientSecret": "8c416dc5-xxxx-xxxx-xxxx-d77069e2a255",
	    "resourceGroup": "test-resource-group",
	    "location": "test-location"
	}
	`
)

func TestConvertAzureCredentialsToAzureCredentialFile(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "azure.toml")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	tempFile.Write([]byte(fakeAzureCredentials))
	os.Setenv("resourceGroup", "test-resource-group")
	os.Setenv("location", "test-location")

	err = convertAzureCredentialsToAzureCredentialFile(tempFile.Name())
	assert.NoError(t, err)

	azureCredentialFilePath, ok := os.LookupEnv("AZURE_CREDENTIAL_FILE")
	assert.True(t, ok)
	assert.Equal(t, azureCredentialFilePath, tempAzureCredentialFilePath)
	defer os.Remove(azureCredentialFilePath)

	azureCredentialFileContent, err := ioutil.ReadFile(azureCredentialFilePath)
	assert.NoError(t, err)
	assert.JSONEq(t, string(azureCredentialFileContent), expectAzureCredentialFileContent)
}
