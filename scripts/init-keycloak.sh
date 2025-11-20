#!/bin/bash

# Install jq if not present
if ! command -v jq &> /dev/null; then
  echo "Installing jq..."
  apk add --no-cache jq
fi

# Wait for Keycloak to be ready
until curl -f http://keycloak:8080/health > /dev/null 2>&1; do
  echo "Waiting for Keycloak to start..."
  sleep 5
done

echo "Keycloak is up. Creating service account..."

# Get admin token
response=$(curl -s -X POST http://keycloak:8080/realms/master/protocol/openid-connect/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=admin" \
  -d "password=admin" \
  -d "grant_type=password" \
  -d "client_id=admin-cli")

echo "Token response: $response"

ADMIN_TOKEN=$(echo "$response" | jq -r '.access_token')

if [ "$ADMIN_TOKEN" == "null" ] || [ -z "$ADMIN_TOKEN" ]; then
  echo "Failed to get admin token"
  exit 1
fi

# Create client for our service
CLIENT_CHECK=$(curl -s -X GET http://keycloak:8080/admin/realms/master/clients \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" | jq -r '.[] | select(.clientId=="omniauth-service") | .id')

if [ -z "$CLIENT_CHECK" ]; then
  echo "Creating client for omniauth-service..."
  create_response=$(curl -s -X POST http://keycloak:8080/admin/realms/master/clients \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
      "clientId": "omniauth-service",
      "name": "Omniauth Service",
      "description": "Service account for Omniauth service",
      "enabled": true,
      "clientAuthenticatorType": "client-secret",
      "secret": "'$KEYCLOAK_CLIENT_SECRET'",
      "publicClient": false,
      "serviceAccountsEnabled": true,
      "standardFlowEnabled": false,
      "implicitFlowEnabled": false,
      "directAccessGrantsEnabled": false,
      "protocol": "openid-connect"
    }')
  echo "Create client response: $create_response"
else
  echo "Client already exists."
fi

echo "Initialization complete."