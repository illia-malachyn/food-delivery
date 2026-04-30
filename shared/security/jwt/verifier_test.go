package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestVerifier_VerifyAccessToken(t *testing.T) {
	t.Parallel()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})
	verifier, err := NewVerifier(string(publicPEM), "food-delivery-auth")
	require.NoError(t, err)

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.RegisteredClaims{
		Issuer:    "food-delivery-auth",
		Subject:   "user-1",
		ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(10 * time.Minute)),
		IssuedAt:  jwtlib.NewNumericDate(time.Now()),
	})
	signedToken, err := token.SignedString(privateKey)
	require.NoError(t, err)

	err = verifier.VerifyAccessToken(signedToken)
	require.NoError(t, err)
}

func TestVerifier_RejectsTokenWithWrongIssuer(t *testing.T) {
	t.Parallel()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})
	verifier, err := NewVerifier(string(publicPEM), "food-delivery-auth")
	require.NoError(t, err)

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.RegisteredClaims{
		Issuer:    "another-issuer",
		Subject:   "user-1",
		ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(10 * time.Minute)),
		IssuedAt:  jwtlib.NewNumericDate(time.Now()),
	})
	signedToken, err := token.SignedString(privateKey)
	require.NoError(t, err)

	err = verifier.VerifyAccessToken(signedToken)
	require.Error(t, err)
}
