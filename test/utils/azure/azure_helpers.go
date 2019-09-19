package azure

import (
	"context"
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
)

type AzureClient struct {
	environment    azure.Environment
	subscriptionID string
	groupsClient   resources.GroupsClient
}

func GetAzureClient(cloud, subscriptionID, clientID, tenantID, clientSecret string) (*AzureClient, error) {
	env, err := azure.EnvironmentFromName(cloud)
	if err != nil {
		return nil, err
	}

	oauthConfig, err := getOAuthConfig(env, subscriptionID, tenantID)
	if err != nil {
		return nil, err
	}

	armSpt, err := adal.NewServicePrincipalToken(*oauthConfig, clientID, clientSecret, env.ServiceManagementEndpoint)
	if err != nil {
		return nil, err
	}

	return getClient(env, subscriptionID, tenantID, armSpt), nil
}

func (az *AzureClient) EnsureResourceGroup(ctx context.Context, name, location string, managedBy *string) (resourceGroup *resources.Group, err error) {
	var tags map[string]*string
	group, err := az.groupsClient.Get(ctx, name)
	if err == nil {
		tags = group.Tags
	}

	response, err := az.groupsClient.CreateOrUpdate(ctx, name, resources.Group{
		Name:      &name,
		Location:  &location,
		ManagedBy: managedBy,
		Tags:      tags,
	})
	if err != nil {
		return &response, err
	}

	return &response, nil
}

func (az *AzureClient) DeleteResourceGroup(ctx context.Context, groupName string) error {
	_, err := az.groupsClient.Get(ctx, groupName)
	if err == nil {
		future, err := az.groupsClient.Delete(ctx, groupName)
		if err != nil {
			return fmt.Errorf("cannot delete resource group %v: %v", groupName, err)
		}
		err = future.WaitForCompletionRef(ctx, az.groupsClient.Client)
		if err != nil {
			// Skip the teardown errors because of https://github.com/Azure/go-autorest/issues/357
			// TODO(feiskyer): fix the issue by upgrading go-autorest version >= v11.3.2.
			log.Printf("Warning: failed to delete resource group %q with error %v", groupName, err)
		}
	}
	return nil
}

func getOAuthConfig(env azure.Environment, subscriptionID, tenantID string) (*adal.OAuthConfig, error) {
	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, err
	}

	return oauthConfig, nil
}

func getClient(env azure.Environment, subscriptionID, tenantID string, armSpt *adal.ServicePrincipalToken) *AzureClient {
	c := &AzureClient{
		environment:    env,
		subscriptionID: subscriptionID,
		groupsClient:   resources.NewGroupsClientWithBaseURI(env.ResourceManagerEndpoint, subscriptionID),
	}

	authorizer := autorest.NewBearerAuthorizer(armSpt)
	c.groupsClient.Authorizer = authorizer

	return c
}
