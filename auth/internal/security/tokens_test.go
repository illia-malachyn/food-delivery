package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/illia-malachyn/food-delivery/auth/internal/config"
)

func newTestManager() TokenManager {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	publicPKIX, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		panic(err)
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicPKIX,
	})

	manager, err := NewJWTManager(config.JWTConfig{
		Issuer:        "test-issuer",
		PrivateKeyPEM: string(privatePEM),
		PublicKeyPEM:  string(publicPEM),
		AccessTTL:     5 * time.Minute,
		RefreshTTL:    30 * time.Minute,
	})
	if err != nil {
		panic(err)
	}
	return manager
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
