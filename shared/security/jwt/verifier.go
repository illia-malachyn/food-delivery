package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type Verifier struct {
	publicKey *rsa.PublicKey
	issuer    string
}

func NewVerifier(publicKeyPEM string, issuer string) (*Verifier, error) {
	publicKey, err := parseRSAPublicKeyPEM(publicKeyPEM)
	if err != nil {
		return nil, err
	}

	return &Verifier{
		publicKey: publicKey,
		issuer:    strings.TrimSpace(issuer),
	}, nil
}

func (v *Verifier) VerifyAccessToken(token string) error {
	claims := &jwt.RegisteredClaims{}
	options := []jwt.ParserOption{jwt.WithValidMethods([]string{"RS256"})}
	if v.issuer != "" {
		options = append(options, jwt.WithIssuer(v.issuer))
	}

	parsedToken, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		return v.publicKey, nil
	}, options...)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !parsedToken.Valid {
		return ErrInvalidToken
	}

	return nil
}

func parseRSAPublicKeyPEM(publicKeyPEM string) (*rsa.PublicKey, error) {
	pemData := strings.TrimSpace(strings.ReplaceAll(publicKeyPEM, `\\n`, "\n"))
	if pemData == "" {
		return nil, fmt.Errorf("empty JWT public key")
	}

	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("invalid PEM block")
	}

	pkixKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		rsaKey, ok := pkixKey.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("public key is not RSA")
		}
		return rsaKey, nil
	}

	rsaKey, pkcs1Err := x509.ParsePKCS1PublicKey(block.Bytes)
	if pkcs1Err == nil {
		return rsaKey, nil
	}

	return nil, fmt.Errorf("failed to parse RSA public key: %w", err)
}
