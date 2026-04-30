package http

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestJWTVerifier_VerifyAccessToken(t *testing.T) {
	t.Parallel()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})
	verifier, err := NewJWTVerifier(string(publicPEM), "food-delivery-auth")
	require.NoError(t, err)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		Issuer:    "food-delivery-auth",
		Subject:   "user-1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	})
	signedToken, err := token.SignedString(privateKey)
	require.NoError(t, err)

	err = verifier.VerifyAccessToken(signedToken)
	require.NoError(t, err)
}

func TestJWTVerifier_RejectsTokenWithWrongIssuer(t *testing.T) {
	t.Parallel()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})
	verifier, err := NewJWTVerifier(string(publicPEM), "food-delivery-auth")
	require.NoError(t, err)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		Issuer:    "another-issuer",
		Subject:   "user-1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	})
	signedToken, err := token.SignedString(privateKey)
	require.NoError(t, err)

	err = verifier.VerifyAccessToken(signedToken)
	require.Error(t, err)
}
