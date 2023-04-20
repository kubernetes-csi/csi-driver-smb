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
	"fmt"
	"html/template"
	"log"
	"os"

	"github.com/kubernetes-csi/csi-driver-smb/test/utils/testutil"

	"github.com/pborman/uuid"
	"github.com/pelletier/go-toml"
)

const (
	AzurePublicCloud            = "AzurePublicCloud"
	AzureChinaCloud             = "AzureChinaCloud"
	ResourceGroupPrefix         = "csi-driver-smb-test-"
	TempAzureCredentialFilePath = "/tmp/azure.json"

	azureCredentialFileTemplate = `{
    "cloud": "{{.Cloud}}",
    "tenantId": "{{.TenantID}}",
    "subscriptionId": "{{.SubscriptionID}}",
    "aadClientId": "{{.AADClientID}}",
    "aadClientSecret": "{{.AADClientSecret}}",
    "resourceGroup": "{{.ResourceGroup}}",
    "location": "{{.Location}}"
}`
	defaultAzurePublicCloudLocation = "eastus2"
	defaultAzureChinaCloudLocation  = "chinaeast2"

	// Env vars
	tenantIDEnvVar        = "AZURE_TENANT_ID"
	subscriptionIDEnvVar  = "AZURE_SUBSCRIPTION_ID"
	aadClientIDEnvVar     = "AZURE_CLIENT_ID"
	aadClientSecretEnvVar = "AZURE_CLIENT_SECRET"
	resourceGroupEnvVar   = "AZURE_RESOURCE_GROUP"
	locationEnvVar        = "AZURE_LOCATION"

	tenantIDChinaEnvVar        = "AZURE_TENANT_ID_CHINA"
	subscriptionIDChinaEnvVar  = "AZURE_SUBSCRIPTION_ID_CHINA"
	aadClientIDChinaEnvVar     = "AZURE_CLIENT_ID_CHINA"
	aadClientSecretChinaEnvVar = "AZURE_CLIENT_SECRET_CHINA"
	resourceGroupChinaEnvVar   = "AZURE_RESOURCE_GROUP_CHINA"
	locationChinaEnvVar        = "AZURE_LOCATION_CHINA"
)

// Config is used in Prow to store Azure credentials
// https://github.com/kubernetes/test-infra/blob/master/kubetest/azure.go#L116-L118
type Config struct {
	Creds FromProw
}

// FromProw is used in Prow to store Azure credentials
// https://github.com/kubernetes/test-infra/blob/master/kubetest/azure.go#L107-L114
type FromProw struct {
	ClientID           string
	ClientSecret       string
	TenantID           string
	SubscriptionID     string
	StorageAccountName string
	StorageAccountKey  string
}

// Credentials is used in Azure File CSI Driver to store Azure credentials
type Credentials struct {
	Cloud           string
	TenantID        string
	SubscriptionID  string
	AADClientID     string
	AADClientSecret string
	ResourceGroup   string
	Location        string
}

// CreateAzureCredentialFile creates a temporary Azure credential file for
// Azure File CSI driver tests and returns the credentials
func CreateAzureCredentialFile(isAzureChinaCloud bool) (*Credentials, error) {
	// Search credentials through env vars first
	var cloud, tenantID, subscriptionID, aadClientID, aadClientSecret, resourceGroup, location string
	if isAzureChinaCloud {
		cloud = AzureChinaCloud
		tenantID = os.Getenv(tenantIDChinaEnvVar)
		subscriptionID = os.Getenv(subscriptionIDChinaEnvVar)
		aadClientID = os.Getenv(aadClientIDChinaEnvVar)
		aadClientSecret = os.Getenv(aadClientSecretChinaEnvVar)
		resourceGroup = os.Getenv(resourceGroupChinaEnvVar)
		location = os.Getenv(locationChinaEnvVar)
	} else {
		cloud = AzurePublicCloud
		tenantID = os.Getenv(tenantIDEnvVar)
		subscriptionID = os.Getenv(subscriptionIDEnvVar)
		aadClientID = os.Getenv(aadClientIDEnvVar)
		aadClientSecret = os.Getenv(aadClientSecretEnvVar)
		resourceGroup = os.Getenv(resourceGroupEnvVar)
		location = os.Getenv(locationEnvVar)
	}

	if resourceGroup == "" {
		resourceGroup = ResourceGroupPrefix + uuid.NewUUID().String()
	}

	if location == "" {
		if isAzureChinaCloud {
			location = defaultAzureChinaCloudLocation
		} else {
			location = defaultAzurePublicCloudLocation
		}
	}

	// Running test locally
	if tenantID != "" && subscriptionID != "" && aadClientID != "" && aadClientSecret != "" {
		return parseAndExecuteTemplate(cloud, tenantID, subscriptionID, aadClientID, aadClientSecret, resourceGroup, location)
	}

	// If the tests are being run in Prow, credentials are not supplied through env vars. Instead, it is supplied
	// through env var AZURE_CREDENTIALS. We need to convert it to AZURE_CREDENTIAL_FILE for sanity, integration and E2E tests
	// https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/cloud-provider-azure/cloud-provider-azure-config.yaml#L5-L6
	if testutil.IsRunningInAzureProw() {
		log.Println("Running in Prow, converting AZURE_CREDENTIALS to AZURE_CREDENTIAL_FILE")
		c, err := getCredentialsFromAzureCredentials(os.Getenv("AZURE_CREDENTIALS"))
		if err != nil {
			return nil, err
		}
		return parseAndExecuteTemplate(cloud, c.TenantID, c.SubscriptionID, c.ClientID, c.ClientSecret, resourceGroup, location)
	}

	return nil, fmt.Errorf("If you are running tests locally, you will need to set the following env vars: $%s, $%s, $%s, $%s, $%s, $%s",
		tenantIDEnvVar, subscriptionIDEnvVar, aadClientIDEnvVar, aadClientSecretEnvVar, resourceGroupEnvVar, locationEnvVar)
}

// DeleteAzureCredentialFile deletes the temporary Azure credential file
func DeleteAzureCredentialFile() error {
	if err := os.Remove(TempAzureCredentialFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error removing %s %v", TempAzureCredentialFilePath, err)
	}

	return nil
}

// getCredentialsFromAzureCredentials parses the azure credentials toml (AZURE_CREDENTIALS)
// in Prow and returns the credential information usable to Azure File CSI driver
func getCredentialsFromAzureCredentials(azureCredentialsPath string) (*FromProw, error) {
	content, err := os.ReadFile(azureCredentialsPath)
	log.Printf("Reading credentials file %v", azureCredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("error reading credentials file %v %v", azureCredentialsPath, err)
	}

	c := Config{}
	if err := toml.Unmarshal(content, &c); err != nil {
		return nil, fmt.Errorf("error parsing credentials file %v %v", azureCredentialsPath, err)
	}

	return &c.Creds, nil
}

// parseAndExecuteTemplate replaces credential placeholders in azureCredentialFileTemplate with actual credentials
func parseAndExecuteTemplate(cloud, tenantID, subscriptionID, aadClientID, aadClientSecret, resourceGroup, location string) (*Credentials, error) {
	t := template.New("AzureCredentialFileTemplate")
	t, err := t.Parse(azureCredentialFileTemplate)
	if err != nil {
		return nil, fmt.Errorf("error parsing azureCredentialFileTemplate %v", err)
	}

	f, err := os.Create(TempAzureCredentialFilePath)
	if err != nil {
		return nil, fmt.Errorf("error creating %s %v", TempAzureCredentialFilePath, err)
	}
	defer f.Close()

	c := Credentials{
		cloud,
		tenantID,
		subscriptionID,
		aadClientID,
		aadClientSecret,
		resourceGroup,
		location,
	}
	err = t.Execute(f, c)
	if err != nil {
		return nil, fmt.Errorf("error executing parsed azure credential file template %v", err)
	}

	return &c, nil
}
