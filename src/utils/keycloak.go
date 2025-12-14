package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/Nerzal/gocloak/v13"
)

// PublicUserData 表示公开的用户信息
type PublicUserData struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

// Keycloak 环境变量键常量
const (
	KeycloakURL          = "KEYCLOAK_URL"
	KeycloakRealm        = "KEYCLOAK_REALM"
	KeycloakClientID     = "KEYCLOAK_CLIENT_ID"
	KeycloakClientSecret = "KEYCLOAK_CLIENT_SECRET"
)

type CloakHelper struct {
	Client       *gocloak.GoCloak
	Realm        string
	ClientID     string
	ClientSecret string
}

func NewCloakHelper() *CloakHelper {
	url := os.Getenv(KeycloakURL)
	if url == "" {
		panic("KEYCLOAK_URL environment variable is not set")
	}

	realm := os.Getenv(KeycloakRealm)
	if realm == "" {
		panic("KEYCLOAK_REALM environment variable is not set")
	}

	clientID := os.Getenv(KeycloakClientID)
	if clientID == "" {
		panic("KEYCLOAK_CLIENT_ID environment variable is not set")
	}

	secret := os.Getenv(KeycloakClientSecret)
	if secret == "" {
		panic("KEYCLOAK_CLIENT_SECRET environment variable is not set")
	}

	return &CloakHelper{
		Client:       gocloak.NewClient(url),
		Realm:        realm,
		ClientID:     clientID,
		ClientSecret: secret,
	}
}

// GetUserProfile fetches a user by ID using the Service Account token
func (s *CloakHelper) GetUserProfile(ctx context.Context, targetUserID string) (*gocloak.User, error) {
	// 1. Login as Service Account (Client Credentials)
	// Optimization Note: You should cache this token and only refresh when it expires.
	token, err := s.Client.LoginClient(ctx, s.ClientID, s.ClientSecret, s.Realm)
	if err != nil {
		return nil, fmt.Errorf("failed to login as service account: %w", err)
	}

	// 2. Use the Admin token to fetch the specific user
	user, err := s.Client.GetUserByID(ctx, token.AccessToken, s.Realm, targetUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user %s: %w", targetUserID, err)
	}

	return user, nil
}
