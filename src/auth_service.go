package main

import (
	"context"

	"github.com/omnsight/omnauth/gen/oauth/v1"
	"github.com/omnsight/omnauth/src/utils"
)

type AuthService struct {
	oauth.UnimplementedAuthServiceServer
	cloakHelper *utils.CloakHelper
}

func NewAuthService(client *utils.CloakHelper) (*AuthService, error) {
	service := &AuthService{
		cloakHelper: client,
	}
	return service, nil
}

func (s *AuthService) GetUser(ctx context.Context, req *oauth.GetUserRequest) (*oauth.GetUserResponse, error) {
	userId, userRoles, err := utils.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	logger := utils.GetLogger(ctx)
	logger.Infof("[%s, %v] requests to view user profile %s", userId, userRoles, req.GetUserId())

	user, err := s.cloakHelper.GetUserProfile(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}

	return &oauth.GetUserResponse{
		User: &oauth.PublicUser{
			Id:        safeStr(user.ID),
			Username:  safeStr(user.Username),
			Firstname: safeStr(user.FirstName),
			Lastname:  safeStr(user.LastName),
			Email:     safeStr(user.Email),
		},
	}, nil
}

// Helper to safely dereference string pointers
func safeStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
