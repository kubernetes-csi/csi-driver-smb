#!/bin/bash

set -euo pipefail

# Default to AzurePublicCloud
cloud='AzurePublicCloud'
if [[ "$#" -gt 0 ]]; then
  cloud="$1"
fi

echo "Creating Azure credential file for $cloud..."
# Copy azure.json template file to $AZURE_CREDENTIAL_FILE
cp hack/template/azure.json "$AZURE_CREDENTIAL_FILE"

# Replace placeholders in the template with environment variables
if [[ "$cloud" == 'AzurePublicCloud' ]] && [[ -v tenantId ]] && [[ -v subscriptionId ]] && [[ -v aadClientId ]] && [[ -v aadClientSecret ]] && [[ -v resourceGroup ]] && [[ -v location ]]; then
  sed -i "s/{{.TenantID}}/$tenantId/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/{{.SubscriptionID}}/$subscriptionId/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/{{.AADClientID}}/$aadClientId/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/{{.AADClientSecret}}/$aadClientSecret/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/{{.ResourceGroup}}/$resourceGroup/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/{{.Location}}/$location/g" "$AZURE_CREDENTIAL_FILE"
elif [[ "$cloud" == 'AzureChinaCloud' ]] && [[ -v tenantId_china ]] && [[ -v subscriptionId_china ]] && [[ -v aadClientId_china ]] && [[ -v aadClientSecret_china ]] && [[ -v resourceGroup_china ]] && [[ -v location_china ]]; then
  sed -i "s/{{.TenantID}}/$tenantId_china/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/{{.SubscriptionID}}/$subscriptionId_china/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/{{.AADClientID}}/$aadClientId_china/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/{{.AADClientSecret}}/$aadClientSecret_china/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/{{.ResourceGroup}}/$resourceGroup_china/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/{{.Location}}/$location_china/g" "$AZURE_CREDENTIAL_FILE"
fi
