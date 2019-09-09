#!/bin/bash

set -euo pipefail

# Default to AzurePublicCloud
cloud='AzurePublicCloud'
if [[ "$#" -gt 0 ]]; then
  cloud="$1"
fi

echo "Creating Azure credential file for $cloud..."
# Copy azure.json template file to $AZURE_CREDENTIAL_FILE
cp hack/azure.json "$AZURE_CREDENTIAL_FILE"

# Replace placeholders in the template with environment variables
if [[ "$cloud" == 'AzurePublicCloud' ]] && [[ -v tenantId ]] && [[ -v subscriptionId ]] && [[ -v aadClientId ]] && [[ -v aadClientSecret ]] && [[ -v resourceGroup ]] && [[ -v location ]]; then
  sed -i "s/tenantId-input/$tenantId/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/subscriptionId-input/$subscriptionId/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/aadClientId-input/$aadClientId/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/aadClientSecret-input/$aadClientSecret/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/resourceGroup-input/$resourceGroup/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/location-input/$location/g" "$AZURE_CREDENTIAL_FILE"
elif [[ "$cloud" == 'AzureChinaCloud' ]] && [[ -v tenantId_china ]] && [[ -v subscriptionId_china ]] && [[ -v aadClientId_china ]] && [[ -v aadClientSecret_china ]] && [[ -v resourceGroup_china ]] && [[ -v location_china ]]; then
  sed -i "s/tenantId-input/$tenantId_china/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/subscriptionId-input/$subscriptionId_china/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/aadClientId-input/$aadClientId_china/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/aadClientSecret-input/$aadClientSecret_china/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/resourceGroup-input/$resourceGroup_china/g" "$AZURE_CREDENTIAL_FILE"
  sed -i "s/location-input/$location_china/g" "$AZURE_CREDENTIAL_FILE"
fi
