/*
Copyright 2017 The Kubernetes Authors.

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

package azurefile

import (
	"context"
	"fmt"
	"os"
	"strings"

	"k8s.io/kubernetes/pkg/cloudprovider/providers/azure"

	"k8s.io/klog"
)

// GetCloudProvider get Azure Cloud Provider
func GetCloudProvider() (*azure.Cloud, error) {
	credFile, ok := os.LookupEnv("AZURE_CREDENTIAL_FILE")
	if ok {
		klog.V(2).Infof("AZURE_CREDENTIAL_FILE env var set as %v", credFile)
	} else {
		credFile = "/etc/kubernetes/azure.json"
		klog.V(2).Infof("use default AZURE_CREDENTIAL_FILE env var: %v", credFile)
	}

	f, err := os.Open(credFile)
	if err != nil {
		klog.Errorf("Failed to load config from file: %s", credFile)
		return nil, fmt.Errorf("Failed to load config from file: %s, cloud not get azure cloud provider", credFile)
	}
	defer f.Close()

	cloud, err := azure.NewCloud(f)
	if err != nil {
		return nil, err
	}

	az, ok := cloud.(*azure.Cloud)
	if !ok || az == nil {
		return nil, fmt.Errorf("failed to get Azure Cloud Provider. GetCloudProvider returned %v instead", cloud)
	}
	return az, nil
}

// GetStorageAccesskey get storage account access key
// todo: use cloud.GetStorageAccesskey func in k8s v1.14.0
func GetStorageAccesskey(cloud *azure.Cloud, account, resourceGroup string) (string, error) {
	ctx, cancel := getContextWithCancel()
	defer cancel()

	result, err := cloud.StorageAccountClient.ListKeys(ctx, resourceGroup, account)
	if err != nil {
		return "", err
	}
	if result.Keys == nil {
		return "", fmt.Errorf("empty keys")
	}

	for _, k := range *result.Keys {
		if k.Value != nil && *k.Value != "" {
			v := *k.Value
			if ind := strings.LastIndex(v, " "); ind >= 0 {
				v = v[(ind + 1):]
			}
			return v, nil
		}
	}
	return "", fmt.Errorf("no valid keys")
}

func getContextWithCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}
