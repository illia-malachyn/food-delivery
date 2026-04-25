package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/illia-malachyn/food-delivery/auth/internal/auth"
	"github.com/illia-malachyn/food-delivery/auth/internal/security"
	"github.com/illia-malachyn/food-delivery/auth/internal/session"
	"github.com/illia-malachyn/food-delivery/auth/internal/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubUserRepo struct {
	createFn      func(ctx context.Context, email, passwordHash string) (user.User, error)
	findByEmailFn func(ctx context.Context, email string) (user.User, error)
	findByIDFn    func(ctx context.Context, userID int64) (user.User, error)
}

func (s stubUserRepo) Create(ctx context.Context, email, passwordHash string) (user.User, error) {
	if s.createFn == nil {
		return user.User{}, errors.New("unexpected Create call")
	}
	return s.createFn(ctx, email, passwordHash)
}

func (s stubUserRepo) FindByEmail(ctx context.Context, email string) (user.User, error) {
	if s.findByEmailFn == nil {
		return user.User{}, errors.New("unexpected FindByEmail call")
	}
	return s.findByEmailFn(ctx, email)
}

func (s stubUserRepo) FindByID(ctx context.Context, userID int64) (user.User, error) {
	if s.findByIDFn == nil {
		return user.User{}, errors.New("unexpected FindByID call")
	}
	return s.findByIDFn(ctx, userID)
}

type stubHasher struct {
	hashFn    func(password string) (string, error)
	compareFn func(hash, password string) error
}

func (s stubHasher) Hash(password string) (string, error) {
	if s.hashFn == nil {
		return "", errors.New("unexpected Hash call")
	}
	return s.hashFn(password)
}

func (s stubHasher) Compare(hash, password string) error {
	if s.compareFn == nil {
		return errors.New("unexpected Compare call")
	}
	return s.compareFn(hash, password)
}

type stubTokenManager struct {
	issueTokenPairFn func(userID int64, email string) (security.IssuedTokenPair, error)
	parseAccessFn    func(token string) (security.AccessPrincipal, error)
	parseRefreshFn   func(token string) (security.RefreshPrincipal, error)
}

func (s stubTokenManager) IssueTokenPair(userID int64, email string) (security.IssuedTokenPair, error) {
	if s.issueTokenPairFn == nil {
		return security.IssuedTokenPair{}, errors.New("unexpected IssueTokenPair call")
	}
	return s.issueTokenPairFn(userID, email)
}

func (s stubTokenManager) ParseAccess(token string) (security.AccessPrincipal, error) {
	if s.parseAccessFn == nil {
		return security.AccessPrincipal{}, errors.New("unexpected ParseAccess call")
	}
	return s.parseAccessFn(token)
}

func (s stubTokenManager) ParseRefresh(token string) (security.RefreshPrincipal, error) {
	if s.parseRefreshFn == nil {
		return security.RefreshPrincipal{}, errors.New("unexpected ParseRefresh call")
	}
	return s.parseRefreshFn(token)
}

type stubRefreshStore struct {
	saveFn          func(ctx context.Context, tokenID string, userID int64, ttl time.Duration) error
	existsForUserFn func(ctx context.Context, tokenID string, userID int64) (bool, error)
	revokeFn        func(ctx context.Context, tokenID string) error
}

var _ session.RefreshStore = (*stubRefreshStore)(nil)

func (s stubRefreshStore) Save(ctx context.Context, tokenID string, userID int64, ttl time.Duration) error {
	if s.saveFn == nil {
		return errors.New("unexpected Save call")
	}
	return s.saveFn(ctx, tokenID, userID, ttl)
}

func (s stubRefreshStore) ExistsForUser(ctx context.Context, tokenID string, userID int64) (bool, error) {
	if s.existsForUserFn == nil {
		return false, errors.New("unexpected ExistsForUser call")
	}
	return s.existsForUserFn(ctx, tokenID, userID)
}

func (s stubRefreshStore) Revoke(ctx context.Context, tokenID string) error {
	if s.revokeFn == nil {
		return errors.New("unexpected Revoke call")
	}
	return s.revokeFn(ctx, tokenID)
}

func TestServiceRegister_Success(t *testing.T) {
	ctx := context.Background()
	refreshTTL := 30 * time.Minute
	var gotEmail, gotPasswordHash string

	svc := auth.NewService(
		stubUserRepo{createFn: func(_ context.Context, email, passwordHash string) (user.User, error) {
			gotEmail = email
			gotPasswordHash = passwordHash
			return user.User{ID: 21, Email: email, PasswordHash: passwordHash}, nil
		}},
		stubHasher{hashFn: func(password string) (string, error) {
			require.Equal(t, "password123", password)
			return "hashed-pass", nil
		}},
		stubTokenManager{issueTokenPairFn: func(userID int64, email string) (security.IssuedTokenPair, error) {
			require.EqualValues(t, 21, userID)
			require.Equal(t, "john@example.com", email)
			return security.IssuedTokenPair{
				AccessToken:    "access-1",
				RefreshToken:   "refresh-1",
				RefreshTokenID: "rt-1",
				TokenType:      "Bearer",
				ExpiresIn:      900,
			}, nil
		}},
		stubRefreshStore{saveFn: func(_ context.Context, tokenID string, userID int64, ttl time.Duration) error {
			require.Equal(t, "rt-1", tokenID)
			require.EqualValues(t, 21, userID)
			require.Equal(t, refreshTTL, ttl)
			return nil
		}},
		refreshTTL,
	)

	result, err := svc.Register(ctx, "  JOHN@EXAMPLE.COM  ", "password123")
	require.NoError(t, err)
	assert.Equal(t, "john@example.com", gotEmail)
	assert.Equal(t, "hashed-pass", gotPasswordHash)
	assert.Equal(t, "access-1", result.AccessToken)
	assert.Equal(t, "refresh-1", result.RefreshToken)
	assert.Equal(t, "Bearer", result.TokenType)
	assert.EqualValues(t, 900, result.ExpiresIn)
}

func TestServiceRegister_MapsEmailAlreadyExists(t *testing.T) {
	svc := auth.NewService(
		stubUserRepo{createFn: func(_ context.Context, _ string, _ string) (user.User, error) {
			return user.User{}, user.ErrEmailUsed
		}},
		stubHasher{hashFn: func(_ string) (string, error) { return "hash", nil }},
		stubTokenManager{},
		stubRefreshStore{},
		time.Hour,
	)

	_, err := svc.Register(context.Background(), "john@example.com", "password123")
	require.ErrorIs(t, err, auth.ErrEmailAlreadyExists)
}

func TestServiceLogin_InvalidCredentialsWhenPasswordMismatch(t *testing.T) {
	svc := auth.NewService(
		stubUserRepo{findByEmailFn: func(_ context.Context, _ string) (user.User, error) {
			return user.User{ID: 7, Email: "john@example.com", PasswordHash: "stored-hash"}, nil
		}},
		stubHasher{compareFn: func(hash, password string) error {
			require.Equal(t, "stored-hash", hash)
			require.Equal(t, "password123", password)
			return errors.New("mismatch")
		}},
		stubTokenManager{},
		stubRefreshStore{},
		time.Hour,
	)

	_, err := svc.Login(context.Background(), "john@example.com", "password123")
	require.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestServiceRefresh_RevokedToken(t *testing.T) {
	svc := auth.NewService(
		stubUserRepo{},
		stubHasher{},
		stubTokenManager{parseRefreshFn: func(token string) (security.RefreshPrincipal, error) {
			require.Equal(t, "refresh-token", token)
			return security.RefreshPrincipal{UserID: 5, TokenID: "rt-old"}, nil
		}},
		stubRefreshStore{existsForUserFn: func(_ context.Context, tokenID string, userID int64) (bool, error) {
			require.Equal(t, "rt-old", tokenID)
			require.EqualValues(t, 5, userID)
			return false, nil
		}},
		time.Hour,
	)

	_, err := svc.Refresh(context.Background(), "refresh-token")
	require.ErrorIs(t, err, auth.ErrRefreshTokenRevoked)
}

func TestServiceRefresh_RotatesAndPersistsNewSession(t *testing.T) {
	steps := make([]string, 0, 4)
	ttl := 45 * time.Minute

	svc := auth.NewService(
		stubUserRepo{findByIDFn: func(_ context.Context, userID int64) (user.User, error) {
			steps = append(steps, "find-user")
			require.EqualValues(t, 42, userID)
			return user.User{ID: 42, Email: "alice@example.com", PasswordHash: "hash"}, nil
		}},
		stubHasher{},
		stubTokenManager{
			parseRefreshFn: func(token string) (security.RefreshPrincipal, error) {
				steps = append(steps, "parse-refresh")
				require.Equal(t, "old-refresh", token)
				return security.RefreshPrincipal{UserID: 42, TokenID: "old-token-id"}, nil
			},
			issueTokenPairFn: func(userID int64, email string) (security.IssuedTokenPair, error) {
				steps = append(steps, "issue-pair")
				require.EqualValues(t, 42, userID)
				require.Equal(t, "alice@example.com", email)
				return security.IssuedTokenPair{
					AccessToken:    "new-access",
					RefreshToken:   "new-refresh",
					RefreshTokenID: "new-token-id",
					TokenType:      "Bearer",
					ExpiresIn:      900,
				}, nil
			},
		},
		stubRefreshStore{
			existsForUserFn: func(_ context.Context, tokenID string, userID int64) (bool, error) {
				steps = append(steps, "exists")
				require.Equal(t, "old-token-id", tokenID)
				require.EqualValues(t, 42, userID)
				return true, nil
			},
			revokeFn: func(_ context.Context, tokenID string) error {
				steps = append(steps, "revoke")
				require.Equal(t, "old-token-id", tokenID)
				return nil
			},
			saveFn: func(_ context.Context, tokenID string, userID int64, gotTTL time.Duration) error {
				steps = append(steps, "save")
				require.Equal(t, "new-token-id", tokenID)
				require.EqualValues(t, 42, userID)
				require.Equal(t, ttl, gotTTL)
				return nil
			},
		},
		ttl,
	)

	result, err := svc.Refresh(context.Background(), "old-refresh")
	require.NoError(t, err)
	assert.Equal(t, "new-access", result.AccessToken)
	assert.Equal(t, "new-refresh", result.RefreshToken)
	assert.Equal(t, []string{"parse-refresh", "exists", "find-user", "revoke", "issue-pair", "save"}, steps)
}

func TestServiceLogout_InvalidRefreshToken(t *testing.T) {
	svc := auth.NewService(
		stubUserRepo{},
		stubHasher{},
		stubTokenManager{parseRefreshFn: func(_ string) (security.RefreshPrincipal, error) {
			return security.RefreshPrincipal{}, security.ErrInvalidRefreshToken
		}},
		stubRefreshStore{},
		time.Hour,
	)

	err := svc.Logout(context.Background(), "bad-token")
	require.ErrorIs(t, err, auth.ErrInvalidRefreshToken)
}
