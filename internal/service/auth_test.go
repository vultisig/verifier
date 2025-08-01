package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/vultisig/verifier/internal/service"
	"github.com/vultisig/verifier/internal/storage"
	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

const testPublicKey = "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

var testLogger = logrus.New()

var _ storage.DatabaseStorage = (*MockDatabaseStorage)(nil)

// MockDatabaseStorage is a mock implementation of storage.DatabaseStorage
type MockDatabaseStorage struct {
	mock.Mock
}

func (m *MockDatabaseStorage) DeleteAllPolicies(ctx context.Context, dbTx pgx.Tx, pluginID types.PluginID, publicKey string) error {
	args := m.Called(ctx, dbTx, pluginID, publicKey)
	return args.Error(0)
}

func (m *MockDatabaseStorage) Pool() *pgxpool.Pool {
	return nil
}

func (m *MockDatabaseStorage) CreateVaultToken(ctx context.Context, token itypes.VaultTokenCreate) (*itypes.VaultToken, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.VaultToken), args.Error(1)
}

func (m *MockDatabaseStorage) GetVaultToken(ctx context.Context, tokenID string) (*itypes.VaultToken, error) {
	args := m.Called(ctx, tokenID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.VaultToken), args.Error(1)
}

func (m *MockDatabaseStorage) RevokeVaultToken(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *MockDatabaseStorage) RevokeAllVaultTokens(ctx context.Context, publicKey string) error {
	args := m.Called(ctx, publicKey)
	return args.Error(0)
}

func (m *MockDatabaseStorage) UpdateVaultTokenLastUsed(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *MockDatabaseStorage) GetActiveVaultTokens(ctx context.Context, publicKey string) ([]itypes.VaultToken, error) {
	args := m.Called(ctx, publicKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]itypes.VaultToken), args.Error(1)
}

// AddPluginPolicySync is a stub to satisfy the DatabaseStorage interface
func (m *MockDatabaseStorage) AddPluginPolicySync(ctx context.Context, tx pgx.Tx, policy itypes.PluginPolicySync) error {
	args := m.Called(ctx, tx, policy)
	return args.Error(0)
}

// Close is a stub to satisfy the DatabaseStorage interface
func (m *MockDatabaseStorage) Close() error {
	return nil
}

// CreatePricing is a stub to satisfy the DatabaseStorage interface
func (m *MockDatabaseStorage) CreatePricing(ctx context.Context, pricingDto types.PricingCreateDto) (*types.Pricing, error) {
	args := m.Called(ctx, pricingDto)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Pricing), args.Error(1)
}

func (m *MockDatabaseStorage) GetPricingByPluginId(ctx context.Context, pluginID types.PluginID) ([]types.Pricing, error) {
	args := m.Called(ctx, pluginID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Pricing), args.Error(1)
}

func (m *MockDatabaseStorage) FindPluginById(ctx context.Context, tx pgx.Tx, pluginID types.PluginID) (*itypes.Plugin, error) {
	args := m.Called(ctx, tx, pluginID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.Plugin), args.Error(1)
}

func (m *MockDatabaseStorage) FindPlugins(ctx context.Context, filters itypes.PluginFilters, take int, skip int, sort string) (*itypes.PluginsPaginatedList, error) {
	args := m.Called(ctx, filters, take, skip, sort)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.PluginsPaginatedList), args.Error(1)
}

func (m *MockDatabaseStorage) GetPluginPolicy(ctx context.Context, id uuid.UUID) (*types.PluginPolicy, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.PluginPolicy), args.Error(1)
}

func (m *MockDatabaseStorage) GetPluginPolicies(ctx context.Context, publicKey string, pluginIds []types.PluginID, includeInactive bool) ([]types.PluginPolicy, error) {
	args := m.Called(ctx, publicKey, pluginIds, includeInactive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.PluginPolicy), args.Error(1)
}

func (m *MockDatabaseStorage) GetAllPluginPolicies(ctx context.Context, publicKey string, pluginID types.PluginID, take int, skip int, includeInactive bool) (*itypes.PluginPolicyPaginatedList, error) {
	args := m.Called(ctx, publicKey, pluginID, take, skip, includeInactive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.PluginPolicyPaginatedList), args.Error(1)
}

func (m *MockDatabaseStorage) DeletePluginPolicyTx(ctx context.Context, dbTx pgx.Tx, id uuid.UUID) error {
	args := m.Called(ctx, dbTx, id)
	return args.Error(0)
}

func (m *MockDatabaseStorage) InsertPluginPolicyTx(ctx context.Context, dbTx pgx.Tx, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	args := m.Called(ctx, dbTx, policy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.PluginPolicy), args.Error(1)
}

func (m *MockDatabaseStorage) UpdatePluginPolicyTx(ctx context.Context, dbTx pgx.Tx, policy types.PluginPolicy) (*types.PluginPolicy, error) {
	args := m.Called(ctx, dbTx, policy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.PluginPolicy), args.Error(1)
}

func (m *MockDatabaseStorage) GetAllFeesByPolicyId(ctx context.Context, policyId uuid.UUID) ([]types.Fee, error) {
	args := m.Called(ctx, policyId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Fee), args.Error(1)
}

func (m *MockDatabaseStorage) MarkFeesCollected(ctx context.Context, collectedAt time.Time, ids []uuid.UUID, txHash string) ([]types.Fee, error) {
	args := m.Called(ctx, collectedAt, ids, txHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Fee), args.Error(1)
}

func (m *MockDatabaseStorage) GetFeesByPublicKey(ctx context.Context, publicKey string, includeCollected bool) ([]types.Fee, error) {
	args := m.Called(ctx, publicKey, includeCollected)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Fee), args.Error(1)
}

func (m *MockDatabaseStorage) GetAllFeesByPublicKey(ctx context.Context, includeCollected bool) ([]types.Fee, error) {
	args := m.Called(ctx, includeCollected)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Fee), args.Error(1)
}

func (m *MockDatabaseStorage) InsertFee(ctx context.Context, dbTx pgx.Tx, fee types.Fee) (*types.Fee, error) {
	args := m.Called(ctx, dbTx, fee)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Fee), args.Error(1)
}

func (m *MockDatabaseStorage) FindPricingById(ctx context.Context, id uuid.UUID) (*types.Pricing, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Pricing), args.Error(1)
}

func (m *MockDatabaseStorage) DeletePricingById(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDatabaseStorage) GetPluginPolicySync(ctx context.Context, id uuid.UUID) (*itypes.PluginPolicySync, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.PluginPolicySync), args.Error(1)
}

func (m *MockDatabaseStorage) DeletePluginPolicySync(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDatabaseStorage) GetUnFinishedPluginPolicySyncs(ctx context.Context) ([]itypes.PluginPolicySync, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]itypes.PluginPolicySync), args.Error(1)
}

func (m *MockDatabaseStorage) UpdatePluginPolicySync(ctx context.Context, dbTx pgx.Tx, policy itypes.PluginPolicySync) error {
	args := m.Called(ctx, dbTx, policy)
	return args.Error(0)
}

func (m *MockDatabaseStorage) AttachTagToPlugin(ctx context.Context, pluginID types.PluginID, tagID string) (*itypes.Plugin, error) {
	args := m.Called(ctx, pluginID, tagID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.Plugin), args.Error(1)
}

func (m *MockDatabaseStorage) CreateRatingForPlugin(ctx context.Context, tx pgx.Tx, pluginID string) error {
	args := m.Called(ctx, tx, pluginID)
	return args.Error(0)
}

func (m *MockDatabaseStorage) CreateReview(ctx context.Context, dbTx pgx.Tx, review itypes.ReviewCreateDto, pluginID string) (string, error) {
	args := m.Called(ctx, dbTx, review, pluginID)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.String(0), args.Error(1)
}

func (m *MockDatabaseStorage) FindRatingByPluginId(ctx context.Context, dbTx pgx.Tx, pluginID string) ([]itypes.PluginRatingDto, error) {
	args := m.Called(ctx, dbTx, pluginID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]itypes.PluginRatingDto), args.Error(1)
}

func (m *MockDatabaseStorage) FindReviewById(ctx context.Context, db pgx.Tx, id string) (*itypes.ReviewDto, error) {
	args := m.Called(ctx, db, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.ReviewDto), args.Error(1)
}

func (m *MockDatabaseStorage) FindReviews(ctx context.Context, pluginId string, take int, skip int, sort string) (itypes.ReviewsDto, error) {
	args := m.Called(ctx, pluginId, take, skip, sort)
	if args.Get(0) == nil {
		return itypes.ReviewsDto{}, args.Error(1)
	}
	return args.Get(0).(itypes.ReviewsDto), args.Error(1)
}

func (m *MockDatabaseStorage) FindTagById(ctx context.Context, id string) (*itypes.Tag, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.Tag), args.Error(1)
}

func (m *MockDatabaseStorage) FindTagByName(ctx context.Context, name string) (*itypes.Tag, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.Tag), args.Error(1)
}

func (m *MockDatabaseStorage) FindTags(ctx context.Context) ([]itypes.Tag, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]itypes.Tag), args.Error(1)
}

func (m *MockDatabaseStorage) UpdateRatingForPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string, reviewRating int) error {
	args := m.Called(ctx, dbTx, pluginId, reviewRating)
	return args.Error(0)
}

func (m *MockDatabaseStorage) WithTransaction(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}
func (m *MockDatabaseStorage) GetAPIKey(ctx context.Context, apiKey string) (*itypes.APIKey, error) {
	args := m.Called(ctx, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.APIKey), args.Error(1)
}

func TestGenerateToken(t *testing.T) {
	testCases := []struct {
		name          string
		secret        string
		publicKey     string
		expectedError bool
	}{
		{
			name:          "Valid secret",
			secret:        "test-secret",
			publicKey:     "test-public-key",
			expectedError: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDatabaseStorage)
			mockDB.On("CreateVaultToken", mock.Anything, mock.Anything).Return(&itypes.VaultToken{
				TokenID:   uuid.New().String(),
				PublicKey: tt.publicKey,
			}, nil)

			auth := service.NewAuthService(tt.secret, mockDB, testLogger)
			token, err := auth.GenerateToken(context.Background(), tt.publicKey)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)
				assert.NotEmpty(t, token)
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	secret := "test-secret-key"
	wrongSecret := "wrong-secret-key"

	testCases := []struct {
		name        string
		setupToken  func() string
		secret      string
		shouldError bool
	}{
		{
			name: "Valid token",
			setupToken: func() string {
				mockDB := new(MockDatabaseStorage)
				mockDB.On("CreateVaultToken", mock.Anything, mock.Anything).Return(&itypes.VaultToken{
					TokenID: "valid-token",
				}, nil)
				mockDB.On("GetVaultToken", mock.Anything, mock.Anything).Return(&itypes.VaultToken{
					TokenID: "valid-token",
				}, nil)
				mockDB.On("UpdateVaultTokenLastUsed", mock.Anything, mock.Anything).Return(nil)

				auth := service.NewAuthService(secret, mockDB, testLogger)
				token, _ := auth.GenerateToken(context.Background(), "test-public-key")
				return token
			},
			secret:      secret,
			shouldError: false,
		},
		{
			name: "Expired token",
			setupToken: func() string {
				// Create a token that's already expired
				claims := &service.Claims{
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString([]byte(secret))
				return tokenString
			},
			secret:      secret,
			shouldError: true,
		},
		{
			name: "Invalid signing method",
			setupToken: func() string {
				claims := &service.Claims{
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
				tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
				return tokenString
			},
			secret:      secret,
			shouldError: true,
		},
		{
			name: "Wrong secret",
			setupToken: func() string {
				mockDB := new(MockDatabaseStorage)
				mockDB.On("CreateVaultToken", mock.Anything, mock.Anything).Return(nil, nil)
				mockDB.On("GetVaultToken", mock.Anything, mock.Anything).Return(nil, nil)
				mockDB.On("UpdateVaultTokenLastUsed", mock.Anything, mock.Anything).Return(nil)

				auth := service.NewAuthService(secret, mockDB, testLogger)
				var token string
				token, _ = auth.GenerateToken(context.Background(), "test-public-key")
				return token
			},
			secret:      wrongSecret,
			shouldError: true,
		},
		{
			name: "Malformed token",
			setupToken: func() string {
				return "not-a-valid-token"
			},
			secret:      secret,
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tokenString := tc.setupToken()
			mockDB := new(MockDatabaseStorage)
			mockDB.On("GetVaultToken", mock.Anything, mock.Anything).Return(&itypes.VaultToken{
				TokenID: "valid-token",
			}, nil)
			mockDB.On("UpdateVaultTokenLastUsed", mock.Anything, mock.Anything).Return(nil)

			authService := service.NewAuthService(tc.secret, mockDB, testLogger)

			claims, err := authService.ValidateToken(context.Background(), tokenString)

			if tc.shouldError {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				assert.True(t, claims.ExpiresAt.After(time.Now()))
			}
		})
	}
}

func TestRefreshToken(t *testing.T) {
	secret := "refresh-test-secret"

	testCases := []struct {
		name        string
		setupToken  func() string
		shouldError bool
	}{
		{
			name: "Valid token refresh",
			setupToken: func() string {
				mockDB := new(MockDatabaseStorage)
				tokenID := uuid.New().String()
				mockDB.On("CreateVaultToken", mock.Anything, mock.Anything).
					Return(&itypes.VaultToken{
						TokenID:   tokenID,
						PublicKey: testPublicKey,
					}, nil)
				mockDB.On("GetVaultToken", mock.Anything, tokenID).
					Return(&itypes.VaultToken{
						TokenID:   tokenID,
						PublicKey: testPublicKey,
					}, nil)
				mockDB.On("UpdateVaultTokenLastUsed", mock.Anything, tokenID).Return(nil)
				mockDB.On("RevokeVaultToken", mock.Anything, tokenID).Return(nil)

				auth := service.NewAuthService(secret, mockDB, testLogger)
				token, _ := auth.GenerateToken(context.Background(), testPublicKey)
				return token
			},
			shouldError: false,
		},
		{
			name: "Expired token refresh",
			setupToken: func() string {
				claims := &service.Claims{
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString([]byte(secret))
				return tokenString
			},
			shouldError: true,
		},
		{
			name: "Invalid token refresh",
			setupToken: func() string {
				return "invalid-token-string"
			},
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tokenString := tc.setupToken()
			mockDB := new(MockDatabaseStorage)

			if !tc.shouldError {
				// For valid token case, set up mock expectations
				tokenID := uuid.New().String()
				mockDB.On("CreateVaultToken", mock.Anything, mock.Anything).
					Return(&itypes.VaultToken{
						TokenID:   tokenID,
						PublicKey: testPublicKey,
					}, nil)
				mockDB.On("GetVaultToken", mock.Anything, mock.Anything).
					Return(&itypes.VaultToken{
						TokenID:   tokenID,
						PublicKey: testPublicKey,
					}, nil)
				mockDB.On("UpdateVaultTokenLastUsed", mock.Anything, mock.Anything).Return(nil)
				mockDB.On("RevokeVaultToken", mock.Anything, mock.Anything).Return(nil)
			}

			authService := service.NewAuthService(secret, mockDB, testLogger)
			newToken, err := authService.RefreshToken(context.Background(), tokenString)

			if tc.shouldError {
				assert.Error(t, err)
				assert.Empty(t, newToken)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, newToken)
				assert.NotEqual(t, tokenString, newToken, "Refreshed token should be different from the original")

				claims, validationErr := authService.ValidateToken(context.Background(), newToken)
				assert.NoError(t, validationErr)
				assert.NotNil(t, claims)
			}
		})
	}
}
