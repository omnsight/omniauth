package main

import (
	"context"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/omnsight/omnauth/gen/oauth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	keycloakURL      = "http://localhost:8080"
	keycloakAdmin    = "admin"
	keycloakPassword = "admin"
	realmName        = "omni"
	clientID         = "omniauth"
	clientSecret     = "omniauth-secret"
	grpcAddr         = "localhost:9092"
)

func TestIntegration(t *testing.T) {
	// Skip if config is not available or explicitly asked to skip (optional)
	// For now we assume env is ready as per user instructions.

	ctx := context.Background()
	client := gocloak.NewClient(keycloakURL)

	// --- 1. Keycloak Setup ---
	t.Log("Setting up Keycloak...")
	token, err := client.LoginAdmin(ctx, keycloakAdmin, keycloakPassword, "master")
	if err != nil {
		t.Fatalf("Failed to login as keycloak admin: %v. Is Keycloak running?", err)
	}

	// Create Users
	adminUser := "admin"
	testUser := "user"
	userPass := "password"
	ensureUser(t, ctx, client, token.AccessToken, adminUser, userPass)
	ensureUser(t, ctx, client, token.AccessToken, testUser, userPass)

	// Assign 'admin' role to adminUser
	users, _ := client.GetUsers(ctx, token.AccessToken, realmName, gocloak.GetUsersParams{Username: gocloak.StringP(adminUser)})
	adminUserID := *users[0].ID

	// Create and Assign 'admin-group' to adminUser
	groupName := "admin-group"
	groups, err := client.GetGroups(ctx, token.AccessToken, realmName, gocloak.GetGroupsParams{Search: gocloak.StringP(groupName)})
	if err != nil {
		t.Fatalf("Failed to get groups: %v", err)
	}

	if len(groups) == 0 {
		t.Fatalf("Failed to find: %v", groupName)
	}
	groupID := *groups[0].ID
	if err = client.AddUserToGroup(ctx, token.AccessToken, realmName, adminUserID, groupID); err != nil {
		t.Fatalf("Failed to add user to admin-group: %v", err)
	}

	// --- 2. gRPC Connection ---
	t.Log("Connecting to gRPC service...")
	// Wait a bit for services to be potentially ready if just started
	time.Sleep(1 * time.Second)

	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to gRPC: %v", err)
	}
	defer conn.Close()

	authClient := oauth.NewAuthServiceClient(conn)

	// Login as admin_user
	userToken, err := client.Login(ctx, clientID, clientSecret, realmName, adminUser, userPass)
	if err != nil {
		t.Fatalf("Failed to login as admin_user: %v", err)
	}
	authCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+userToken.AccessToken))

	// --- 1. get user ---
	// Get test user ID to verify against
	tUsers, err := client.GetUsers(ctx, token.AccessToken, realmName, gocloak.GetUsersParams{Username: gocloak.StringP(testUser)})
	if err != nil {
		t.Fatalf("Failed to get test user info: %v", err)
	}
	if len(tUsers) == 0 {
		t.Fatalf("Test user %s not found", testUser)
	}
	testUserID := *tUsers[0].ID

	resp, err := authClient.GetUser(authCtx, &oauth.GetUserRequest{
		UserId: testUserID,
	})
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	t.Logf("User: %v", resp.User)

	// Verify response
	if resp.User.Id != testUserID {
		t.Errorf("Mismatch in User ID: expected %s, got %s", testUserID, resp.User.Id)
	}
	if resp.User.Username != testUser {
		t.Errorf("Mismatch in Username: expected %s, got %s", testUser, resp.User.Username)
	}

	t.Log("Integration test completed successfully.")
}

func ensureUser(t *testing.T, ctx context.Context, client *gocloak.GoCloak, token, username, password string) {
	user := gocloak.User{
		Username:      gocloak.StringP(username),
		FirstName:     gocloak.StringP(username),
		LastName:      gocloak.StringP("User"),
		Email:         gocloak.StringP(username + "@example.com"),
		Enabled:       gocloak.BoolP(true),
		EmailVerified: gocloak.BoolP(true),
	}
	id, err := client.CreateUser(ctx, token, realmName, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	if err := client.SetPassword(ctx, token, id, realmName, password, false); err != nil {
		t.Fatalf("Failed to set password: %v", err)
	}
	// Fetch again to get the ID
	createdUsers, err := client.GetUsers(ctx, token, realmName, gocloak.GetUsersParams{Username: gocloak.StringP(username)})
	if err != nil || len(createdUsers) == 0 {
		t.Fatalf("Failed to retrieve created user: %v", err)
	}
	userToUpdate := *createdUsers[0]

	// Clear required actions and ensure email is verified to avoid "Account is not fully set up"
	emptyActions := []string{}
	userToUpdate.RequiredActions = &emptyActions
	userToUpdate.EmailVerified = gocloak.BoolP(true)
	userToUpdate.FirstName = gocloak.StringP(username)
	userToUpdate.LastName = gocloak.StringP("User")
	userToUpdate.Email = gocloak.StringP(username + "@example.com")

	if err := client.UpdateUser(ctx, token, realmName, userToUpdate); err != nil {
		t.Fatalf("Failed to update user actions: %v", err)
	}

	t.Logf("Ensured user: %s (ID: %s, Email: %s)", *userToUpdate.Username, *userToUpdate.ID, *userToUpdate.Email)
}
