#!/bin/sh

echo "Using Keycloak configuration:"
echo "  Realm: $KEYCLOAK_REALM"
echo "  Client ID: $KEYCLOAK_CLIENT_ID"
echo "  Client Secret: $KEYCLOAK_CLIENT_SECRET"

#####################################################################################################################
# Get admin token
#####################################################################################################################
response=$(curl -s -X POST $KEYCLOAK_URL/realms/master/protocol/openid-connect/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=admin" \
  -d "password=admin" \
  -d "grant_type=password" \
  -d "client_id=admin-cli")
echo "Token response: $response"
ADMIN_TOKEN=$(echo $response | jq -r '.access_token')

if [ "$ADMIN_TOKEN" = "" ]; then
  echo "Failed to get admin token"
  exit 1
fi

echo "  Admin token: $ADMIN_TOKEN"

#####################################################################################################################
# Create realm
#####################################################################################################################
REALM_CHECK=$(curl -s -X GET $KEYCLOAK_URL/admin/realms/$KEYCLOAK_REALM \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json")
echo "Realm check response: $REALM_CHECK"
EXISTING_REALM_NAME=$(echo "$REALM_CHECK" | jq -r '.realm')

if [ "$EXISTING_REALM_NAME" == "$KEYCLOAK_REALM" ]; then
    echo "❌ Realm '$KEYCLOAK_REALM' exists. Skipping creation."
    exit 1
else
    echo "Realm '$KEYCLOAK_REALM' not found. Creating it now..."
    create_realm_response=$(curl -s -X POST http://keycloak:8080/admin/realms \
      -H "Authorization: Bearer $ADMIN_TOKEN" \
      -H "Content-Type: application/json" \
      -d '{
        "realm": "'$KEYCLOAK_REALM'",
        "enabled": true
      }')
    echo "✅ Create realm response: $create_realm_response"
fi

#####################################################################################################################
# Create client
#####################################################################################################################
CLIENT_LIST=$(curl -s -X GET "http://keycloak:8080/admin/realms/$KEYCLOAK_REALM/clients?clientId=$KEYCLOAK_CLIENT_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json")
echo "Client list response: $CLIENT_LIST"
CLIENT_COUNT=$(echo "$CLIENT_LIST" | jq length)

if [ "$CLIENT_COUNT" -eq 0 ]; then
  echo "Creating client for $KEYCLOAK_CLIENT_ID..."
  create_response=$(curl -s -X POST http://keycloak:8080/admin/realms/$KEYCLOAK_REALM/clients \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
      "clientId": "'$KEYCLOAK_CLIENT_ID'",
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
  echo "✅ Create client response: $create_response"
else
  echo "❌ Client already exists. Updating client secret..."
  exit 1
fi

#####################################################################################################################
# Set up client permission
#####################################################################################################################

# 1. Get the Internal ID of your new Client (Omniauth Service)
CLIENT_UUID=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$KEYCLOAK_URL/admin/realms/$KEYCLOAK_REALM/clients?clientId=$KEYCLOAK_CLIENT_ID" | jq -r '.[0].id')

# 2. Get the User ID associated with the Service Account
# (Service Accounts are technically "Users" in Keycloak)
SA_USER_ID=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$KEYCLOAK_URL/admin/realms/$KEYCLOAK_REALM/clients/$CLIENT_UUID/service-account-user" | jq -r '.id')

# 3. Get the Internal ID of the "realm-management" client
# (This client holds the admin roles like view-users)
REALM_MGMT_ID=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$KEYCLOAK_URL/admin/realms/$KEYCLOAK_REALM/clients?clientId=realm-management" | jq -r '.[0].id')

# 4. Get the specific "view-users" role definition
VIEW_USERS_ROLE=$(curl -s -H "Authorization: Bearer $ADMIN_TOKEN" \
  "$KEYCLOAK_URL/admin/realms/$KEYCLOAK_REALM/clients/$REALM_MGMT_ID/roles/manage-users")

# 5. Map the role to the Service Account User
curl -s -X POST "$KEYCLOAK_URL/admin/realms/$KEYCLOAK_REALM/users/$SA_USER_ID/role-mappings/clients/$REALM_MGMT_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "[$VIEW_USERS_ROLE]"

echo "✅ Service account granted 'view-users' permission."

#####################################################################################################################
echo "✅ Initialization complete."
