package security

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/illia-malachyn/food-delivery/auth/internal/config"
)

var (
	ErrInvalidAccessToken  = errors.New("invalid access token")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

type AccessPrincipal struct {
	UserID int64
	Email  string
}

type RefreshPrincipal struct {
	UserID  int64
	TokenID string
}

type IssuedTokenPair struct {
	AccessToken    string
	RefreshToken   string
	RefreshTokenID string
	TokenType      string
	ExpiresIn      int64
}

type TokenManager interface {
	IssueTokenPair(userID int64, email string) (IssuedTokenPair, error)
	ParseAccess(token string) (AccessPrincipal, error)
	ParseRefresh(token string) (RefreshPrincipal, error)
}

type jwtManager struct {
	issuer        string
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

type accessClaims struct {
	Email string `json:"email"`
	Type  string `json:"typ"`
	jwt.RegisteredClaims
}

type refreshClaims struct {
	Type string `json:"typ"`
	jwt.RegisteredClaims
}

func NewJWTManager(cfg config.JWTConfig) TokenManager {
	return &jwtManager{
		issuer:        cfg.Issuer,
		accessSecret:  []byte(cfg.AccessSecret),
		refreshSecret: []byte(cfg.RefreshSecret),
		accessTTL:     cfg.AccessTTL,
		refreshTTL:    cfg.RefreshTTL,
	}
}

func (m *jwtManager) IssueTokenPair(userID int64, email string) (IssuedTokenPair, error) {
	now := time.Now().UTC()
	userIDStr := strconv.FormatInt(userID, 10)
	refreshTokenID := uuid.NewString()

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims{
		Email: email,
		Type:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   userIDStr,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
	}).SignedString(m.accessSecret)
	if err != nil {
		return IssuedTokenPair{}, err
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims{
		Type: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        refreshTokenID,
			Subject:   userIDStr,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTTL)),
		},
	}).SignedString(m.refreshSecret)
	if err != nil {
		return IssuedTokenPair{}, err
	}

	return IssuedTokenPair{
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		RefreshTokenID: refreshTokenID,
		TokenType:      "Bearer",
		ExpiresIn:      int64(m.accessTTL.Seconds()),
	}, nil
}

func (m *jwtManager) ParseAccess(rawToken string) (AccessPrincipal, error) {
	token := strings.TrimSpace(rawToken)
	if token == "" {
		return AccessPrincipal{}, ErrInvalidAccessToken
	}

	parsed, err := jwt.ParseWithClaims(token, &accessClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.accessSecret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return AccessPrincipal{}, ErrInvalidAccessToken
	}

	claims, ok := parsed.Claims.(*accessClaims)
	if !ok || !parsed.Valid || claims.Type != "access" {
		return AccessPrincipal{}, ErrInvalidAccessToken
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return AccessPrincipal{}, ErrInvalidAccessToken
	}

	return AccessPrincipal{UserID: userID, Email: claims.Email}, nil
}

func (m *jwtManager) ParseRefresh(rawToken string) (RefreshPrincipal, error) {
	token := strings.TrimSpace(rawToken)
	if token == "" {
		return RefreshPrincipal{}, ErrInvalidRefreshToken
	}

	parsed, err := jwt.ParseWithClaims(token, &refreshClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.refreshSecret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return RefreshPrincipal{}, ErrInvalidRefreshToken
	}

	claims, ok := parsed.Claims.(*refreshClaims)
	if !ok || !parsed.Valid || claims.Type != "refresh" || claims.ID == "" {
		return RefreshPrincipal{}, ErrInvalidRefreshToken
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return RefreshPrincipal{}, ErrInvalidRefreshToken
	}

	return RefreshPrincipal{UserID: userID, TokenID: claims.ID}, nil
}
