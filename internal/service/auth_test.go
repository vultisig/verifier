package service_test

import (
	"testing"
	"time"

	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"

	"github.com/vultisig/verifier/internal/service"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const testPublicKey = "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

var testLogger = logrus.New()

// MockDatabaseStorage is a mock implementation of storage.DatabaseStorage
type MockDatabaseStorage struct {
	mock.Mock
}

func (m *MockDatabaseStorage) Pool() *pgxpool.Pool {
	return nil
}

func (m *MockDatabaseStorage) FindUserById(ctx context.Context, userId string) (*itypes.User, error) {
	args := m.Called(ctx, userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.User), args.Error(1)
}

func (m *MockDatabaseStorage) FindUserByName(ctx context.Context, username string) (*itypes.UserWithPassword, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.UserWithPassword), args.Error(1)
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

// CountTransactions is a stub to satisfy the DatabaseStorage interface
func (m *MockDatabaseStorage) CountTransactions(ctx context.Context, policyID uuid.UUID, status itypes.TransactionStatus, txType string) (int64, error) {
	args := m.Called(ctx, policyID, status, txType)
	if args.Get(0) == nil {
		return 0, args.Error(1)
	}
	return args.Get(0).(int64), args.Error(1)
}

// CreatePlugin is a stub to satisfy the DatabaseStorage interface
func (m *MockDatabaseStorage) CreatePlugin(ctx context.Context, tx pgx.Tx, pluginDto itypes.PluginCreateDto) (string, error) {
	args := m.Called(ctx, tx, pluginDto)
	return args.String(0), args.Error(1)
}

// CreatePricing is a stub to satisfy the DatabaseStorage interface
func (m *MockDatabaseStorage) CreatePricing(ctx context.Context, pricingDto itypes.PricingCreateDto) (*itypes.Pricing, error) {
	args := m.Called(ctx, pricingDto)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.Pricing), args.Error(1)
}

// CreateTransactionHistory is a stub to satisfy the DatabaseStorage interface
func (m *MockDatabaseStorage) CreateTransactionHistory(ctx context.Context, tx itypes.TransactionHistory) (uuid.UUID, error) {
	args := m.Called(ctx, tx)
	if args.Get(0) == nil {
		return uuid.Nil, args.Error(1)
	}
	return args.Get(0).(uuid.UUID), args.Error(1)
}

// CreateTransactionHistoryTx is a stub to satisfy the DatabaseStorage interface
func (m *MockDatabaseStorage) CreateTransactionHistoryTx(ctx context.Context, dbTx pgx.Tx, tx itypes.TransactionHistory) (uuid.UUID, error) {
	args := m.Called(ctx, dbTx, tx)
	if args.Get(0) == nil {
		return uuid.Nil, args.Error(1)
	}
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockDatabaseStorage) DeletePluginById(ctx context.Context, id types.PluginID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDatabaseStorage) FindPluginById(ctx context.Context, tx pgx.Tx, pluginID types.PluginID) (*itypes.Plugin, error) {
	args := m.Called(ctx, tx, pluginID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.Plugin), args.Error(1)
}

func (m *MockDatabaseStorage) FindPlugins(ctx context.Context, filters itypes.PluginFilters, page, pageSize int, sortBy string) (itypes.PluginsPaginatedList, error) {
	args := m.Called(ctx, filters, page, pageSize, sortBy)
	if args.Get(0) == nil {
		return itypes.PluginsPaginatedList{}, args.Error(1)
	}
	return args.Get(0).(itypes.PluginsPaginatedList), args.Error(1)
}

func (m *MockDatabaseStorage) UpdatePlugin(ctx context.Context, pluginID types.PluginID, updates itypes.PluginUpdateDto) (*itypes.Plugin, error) {
	args := m.Called(ctx, pluginID, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.Plugin), args.Error(1)
}

func (m *MockDatabaseStorage) GetPluginPolicy(ctx context.Context, id uuid.UUID) (types.PluginPolicy, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return types.PluginPolicy{}, args.Error(1)
	}
	return args.Get(0).(types.PluginPolicy), args.Error(1)
}

func (m *MockDatabaseStorage) GetAllPluginPolicies(ctx context.Context, pluginType string, publicKeyEcdsa string, take int, skip int) (itypes.PluginPolicyPaginatedList, error) {
	args := m.Called(ctx, pluginType, publicKeyEcdsa, take, skip)
	if args.Get(0) == nil {
		return itypes.PluginPolicyPaginatedList{}, args.Error(1)
	}
	return args.Get(0).(itypes.PluginPolicyPaginatedList), args.Error(1)
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

func (m *MockDatabaseStorage) FindPricingById(ctx context.Context, id uuid.UUID) (*itypes.Pricing, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.Pricing), args.Error(1)
}

func (m *MockDatabaseStorage) DeletePricingById(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDatabaseStorage) UpdateTransactionStatusTx(ctx context.Context, dbTx pgx.Tx, txID uuid.UUID, status itypes.TransactionStatus, metadata map[string]interface{}) error {
	args := m.Called(ctx, dbTx, txID, status, metadata)
	return args.Error(0)
}

func (m *MockDatabaseStorage) GetTransactionHistory(ctx context.Context, policyID uuid.UUID, transactionType string, take int, skip int) ([]itypes.TransactionHistory, error) {
	args := m.Called(ctx, policyID, transactionType, take, skip)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]itypes.TransactionHistory), args.Error(1)
}

func (m *MockDatabaseStorage) GetTransactionByHash(ctx context.Context, txHash string) (*itypes.TransactionHistory, error) {
	args := m.Called(ctx, txHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.TransactionHistory), args.Error(1)
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

func (m *MockDatabaseStorage) UpdateTransactionStatus(ctx context.Context, txID uuid.UUID, status itypes.TransactionStatus, metadata map[string]interface{}) error {
	args := m.Called(ctx, txID, status, metadata)
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

func (m *MockDatabaseStorage) CreateReview(ctx context.Context, review itypes.ReviewCreateDto, pluginID string) (string, error) {
	args := m.Called(ctx, review, pluginID)
	return args.String(0), args.Error(1)
}

func (m *MockDatabaseStorage) CreateTag(ctx context.Context, tag itypes.CreateTagDto) (*itypes.Tag, error) {
	args := m.Called(ctx, tag)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.Tag), args.Error(1)
}

func (m *MockDatabaseStorage) DetachTagFromPlugin(ctx context.Context, pluginID types.PluginID, tagID string) (*itypes.Plugin, error) {
	args := m.Called(ctx, pluginID, tagID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.Plugin), args.Error(1)
}

func (m *MockDatabaseStorage) FindCategories(ctx context.Context) ([]itypes.Category, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]itypes.Category), args.Error(1)
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
					TokenID:   "valid-token",
					IsRevoked: false,
				}, nil)
				mockDB.On("GetVaultToken", mock.Anything, mock.Anything).Return(&itypes.VaultToken{
					TokenID:   "valid-token",
					IsRevoked: false,
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
				TokenID:   "valid-token",
				IsRevoked: false,
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
						IsRevoked: false,
					}, nil)
				mockDB.On("GetVaultToken", mock.Anything, tokenID).
					Return(&itypes.VaultToken{
						TokenID:   tokenID,
						PublicKey: testPublicKey,
						IsRevoked: false,
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
						IsRevoked: false,
					}, nil)
				mockDB.On("GetVaultToken", mock.Anything, mock.Anything).
					Return(&itypes.VaultToken{
						TokenID:   tokenID,
						PublicKey: testPublicKey,
						IsRevoked: false,
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
