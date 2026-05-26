package service

import (
	"testing"

	"LanLine/internal/middleware"
)

func TestChangePasswordRevokesExistingTokens(t *testing.T) {
	setupChatSecurityTest(t)
	authSvc := &AuthService{JWTSecret: "auth-secret", JWTExpireHrs: 1}

	user, err := authSvc.Register("change_pw_user", "old-password", "改密用户")
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	token, _, err := authSvc.Login("change_pw_user", "old-password")
	if err != nil {
		t.Fatalf("login user: %v", err)
	}
	if _, err := middleware.ParseToken(token, authSvc.JWTSecret); err != nil {
		t.Fatalf("expected token to be valid before password change: %v", err)
	}

	if err := authSvc.ChangePassword(user.ID, "old-password", "new-password"); err != nil {
		t.Fatalf("change password: %v", err)
	}
	if _, err := middleware.ParseToken(token, authSvc.JWTSecret); err == nil {
		t.Fatal("expected old token to be invalid after password change")
	}
}

func TestRevokeTokenInvalidatesCurrentToken(t *testing.T) {
	setupChatSecurityTest(t)
	authSvc := &AuthService{JWTSecret: "revoke-secret", JWTExpireHrs: 1}

	if _, err := authSvc.Register("revoke_user", "password", "吊销用户"); err != nil {
		t.Fatalf("register user: %v", err)
	}
	token, _, err := authSvc.Login("revoke_user", "password")
	if err != nil {
		t.Fatalf("login user: %v", err)
	}

	if err := authSvc.RevokeToken(token); err != nil {
		t.Fatalf("revoke token: %v", err)
	}
	if _, err := middleware.ParseToken(token, authSvc.JWTSecret); err == nil {
		t.Fatal("expected revoked token to be invalid")
	}
}
