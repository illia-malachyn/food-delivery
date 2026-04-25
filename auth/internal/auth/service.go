package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/illia-malachyn/food-delivery/auth/internal/security"
	"github.com/illia-malachyn/food-delivery/auth/internal/session"
	"github.com/illia-malachyn/food-delivery/auth/internal/user"
)

type Service struct {
	users        user.Repository
	hasher       security.PasswordHasher
	tokens       security.TokenManager
	refreshStore session.RefreshStore
	refreshTTL   time.Duration
}

type AuthResult struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int64
}

func NewService(
	users user.Repository,
	hasher security.PasswordHasher,
	tokens security.TokenManager,
	refreshStore session.RefreshStore,
	refreshTTL time.Duration,
) *Service {
	return &Service{
		users:        users,
		hasher:       hasher,
		tokens:       tokens,
		refreshStore: refreshStore,
		refreshTTL:   refreshTTL,
	}
}

func (s *Service) Register(ctx context.Context, email, password string) (AuthResult, error) {
	email = normalizeEmail(email)
	if err := validateCredentials(email, password); err != nil {
		return AuthResult{}, err
	}

	passwordHash, err := s.hasher.Hash(password)
	if err != nil {
		return AuthResult{}, err
	}

	createdUser, err := s.users.Create(ctx, email, passwordHash)
	if err != nil {
		if errors.Is(err, user.ErrEmailUsed) {
			return AuthResult{}, ErrEmailAlreadyExists
		}
		return AuthResult{}, err
	}

	return s.issueAndPersist(ctx, createdUser.ID, createdUser.Email)
}

func (s *Service) Login(ctx context.Context, email, password string) (AuthResult, error) {
	email = normalizeEmail(email)
	if err := validateCredentials(email, password); err != nil {
		return AuthResult{}, err
	}

	storedUser, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return AuthResult{}, ErrInvalidCredentials
		}
		return AuthResult{}, err
	}

	if err := s.hasher.Compare(storedUser.PasswordHash, password); err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}

	return s.issueAndPersist(ctx, storedUser.ID, storedUser.Email)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (AuthResult, error) {
	refreshPrincipal, err := s.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return AuthResult{}, ErrInvalidRefreshToken
	}

	exists, err := s.refreshStore.ExistsForUser(ctx, refreshPrincipal.TokenID, refreshPrincipal.UserID)
	if err != nil {
		return AuthResult{}, err
	}
	if !exists {
		return AuthResult{}, ErrRefreshTokenRevoked
	}

	storedUser, err := s.users.FindByID(ctx, refreshPrincipal.UserID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return AuthResult{}, ErrInvalidRefreshToken
		}
		return AuthResult{}, err
	}

	if err := s.refreshStore.Revoke(ctx, refreshPrincipal.TokenID); err != nil {
		return AuthResult{}, err
	}

	return s.issueAndPersist(ctx, storedUser.ID, storedUser.Email)
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	refreshPrincipal, err := s.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return ErrInvalidRefreshToken
	}

	if err := s.refreshStore.Revoke(ctx, refreshPrincipal.TokenID); err != nil {
		return err
	}

	return nil
}

func (s *Service) issueAndPersist(ctx context.Context, userID int64, email string) (AuthResult, error) {
	issuedPair, err := s.tokens.IssueTokenPair(userID, email)
	if err != nil {
		return AuthResult{}, err
	}

	if err := s.refreshStore.Save(ctx, issuedPair.RefreshTokenID, userID, s.refreshTTL); err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		AccessToken:  issuedPair.AccessToken,
		RefreshToken: issuedPair.RefreshToken,
		TokenType:    issuedPair.TokenType,
		ExpiresIn:    issuedPair.ExpiresIn,
	}, nil
}

func validateCredentials(email, password string) error {
	if email == "" || !strings.Contains(email, "@") {
		return ErrInvalidEmail
	}
	if len(password) < 8 {
		return ErrInvalidPassword
	}
	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
