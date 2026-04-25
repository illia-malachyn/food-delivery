package security

import (
	"testing"
	"time"

	"github.com/illia-malachyn/food-delivery/auth/internal/config"
)

func newTestManager() TokenManager {
	return NewJWTManager(config.JWTConfig{
		Issuer:        "test-issuer",
		AccessSecret:  "test-access-secret-very-long",
		RefreshSecret: "test-refresh-secret-very-long",
		AccessTTL:     5 * time.Minute,
		RefreshTTL:    30 * time.Minute,
	})
}

func TestJWTManagerIssueAndParse(t *testing.T) {
	manager := newTestManager()

	pair, err := manager.IssueTokenPair(42, "alice@example.com")
	if err != nil {
		t.Fatalf("IssueTokenPair() error = %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" || pair.RefreshTokenID == "" {
		t.Fatal("issued pair must contain non-empty tokens and refresh token ID")
	}

	accessPrincipal, err := manager.ParseAccess(pair.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccess() error = %v", err)
	}
	if accessPrincipal.UserID != 42 || accessPrincipal.Email != "alice@example.com" {
		t.Fatalf("unexpected access principal: %+v", accessPrincipal)
	}

	refreshPrincipal, err := manager.ParseRefresh(pair.RefreshToken)
	if err != nil {
		t.Fatalf("ParseRefresh() error = %v", err)
	}
	if refreshPrincipal.UserID != 42 {
		t.Fatalf("unexpected refresh user id: %d", refreshPrincipal.UserID)
	}
	if refreshPrincipal.TokenID != pair.RefreshTokenID {
		t.Fatalf("unexpected refresh token ID: %s", refreshPrincipal.TokenID)
	}
}

func TestJWTManagerRejectsWrongTokenType(t *testing.T) {
	manager := newTestManager()

	pair, err := manager.IssueTokenPair(7, "bob@example.com")
	if err != nil {
		t.Fatalf("IssueTokenPair() error = %v", err)
	}

	if _, err := manager.ParseAccess(pair.RefreshToken); err == nil {
		t.Fatal("ParseAccess() expected error for refresh token")
	}
	if _, err := manager.ParseRefresh(pair.AccessToken); err == nil {
		t.Fatal("ParseRefresh() expected error for access token")
	}
}
