package credentials

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"

	"github.com/BurntSushi/toml"
	uuid "github.com/satori/go.uuid"
	"k8s.io/klog"
)

const (
	tempAzureCredentialFilePath = "/tmp/azure.json"
)

// AzureCredentialsConfig is used in Prow to store Azure credentials
type AzureCredentialsConfig struct {
	Creds AzureCredentialsFromProw
}

// AzureCredentialsFromProw is used in Prow to store Azure credentials
type AzureCredentialsFromProw struct {
	ClientID           string
	ClientSecret       string
	TenantID           string
	SubscriptionID     string
	StorageAccountName string
	StorageAccountKey  string
}

// AzureCredentials is used in Azure File CSI Driver to store Azure credentials
type AzureCredentials struct {
	TenantID        string
	SubscriptionID  string
	AADClientID     string
	AADClientSecret string
	ResourceGroup   string
	Location        string
}

func Get() (*AzureCredentials, error) {
	// Need to obtain credentials from env var AZURE_CREDENTIALS and convert
	// it to AZURE_CREDENTIAL_FILE for sanity and integration tests if we are testing in Prow
	// https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/cloud-provider-azure/cloud-provider-azure-config.yaml#L5
	if azureCredentials, ok := os.LookupEnv("AZURE_CREDENTIALS"); ok {
		klog.V(2).Infof("Running in Prow, converting AZURE_CREDENTIALS to AZURE_CREDENTIAL_FILE")
		config, err := convertAzureCredentialsToAzureCredentialFile(azureCredentials)
		if err != nil {
			return nil, err
		}
		return config, nil
	}

	return nil, fmt.Errorf("$AZURE_CREDENTIALS is not set")
}

// convertAzureCredentialsToAzureCredentialFile converts azure credentials (AZURE_CREDENTIALS)
// in Prow to azure.json (AZURE_CREDENTIAL_FILE) using Go template
func convertAzureCredentialsToAzureCredentialFile(azureCredentialsPath string) (*AzureCredentials, error) {
	content, err := ioutil.ReadFile(azureCredentialsPath)
	klog.V(2).Infof("Reading credentials file %v", azureCredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("error reading credentials file %v %v", azureCredentialsPath, err)
	}

	c := AzureCredentialsConfig{}
	if err := toml.Unmarshal(content, &c); err != nil {
		return nil, fmt.Errorf("error parsing credentials file %v %v", azureCredentialsPath, err)
	}

	// Get resource group name and location from environment variables
	resourceGroup, ok := os.LookupEnv("resourceGroup")
	if !ok {
		// https://github.com/kubernetes/test-infra/blob/master/kubetest/azure.go#L341
		resourceGroup = "kubetest-" + uuid.NewV1().String()
	}
	location, ok := os.LookupEnv("location")
	if !ok {
		location = "eastus2"
	}

	t, err := template.ParseFiles("../../hack/template/azure.json")
	if err != nil {
		return nil, err
	}

	f, err := os.Create(tempAzureCredentialFilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Apply credentials to placeholders in the template
	config := AzureCredentials{
		c.Creds.TenantID,
		c.Creds.SubscriptionID,
		c.Creds.ClientID,
		c.Creds.ClientSecret,
		resourceGroup,
		location,
	}
	err = t.Execute(f, config)
	if err != nil {
		return nil, fmt.Errorf("error executing parsed azure credential file tempalte %v", err)
	}

	// Set the environment variable AZURE_CREDENTIAL_FILE for next steps
	os.Setenv("AZURE_CREDENTIAL_FILE", tempAzureCredentialFilePath)

	return &config, nil
}
