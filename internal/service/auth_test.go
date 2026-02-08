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
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/rpc"

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

func (m *MockDatabaseStorage) DeleteAllPolicies(ctx context.Context, dbTx pgx.Tx, pluginID string, publicKey string) error {
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

func (m *MockDatabaseStorage) GetPricingByPluginId(ctx context.Context, pluginID string) ([]types.Pricing, error) {
	args := m.Called(ctx, pluginID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Pricing), args.Error(1)
}

func (m *MockDatabaseStorage) FindPluginById(ctx context.Context, tx pgx.Tx, pluginID string) (*itypes.Plugin, error) {
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

func (m *MockDatabaseStorage) GetPluginInstallationsCount(ctx context.Context, pluginID string) (itypes.PluginTotalCount, error) {
	args := m.Called(ctx, pluginID)
	if args.Get(0) == nil {
		return itypes.PluginTotalCount{}, args.Error(1)
	}
	return args.Get(0).(itypes.PluginTotalCount), args.Error(1)
}

func (m *MockDatabaseStorage) GetPluginPolicies(ctx context.Context, publicKey string, pluginIds []string, includeInactive bool) ([]types.PluginPolicy, error) {
	args := m.Called(ctx, publicKey, pluginIds, includeInactive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.PluginPolicy), args.Error(1)
}

func (m *MockDatabaseStorage) GetAllPluginPolicies(ctx context.Context, publicKey string, pluginID string, take int, skip int, activeFilter *bool) (*itypes.PluginPolicyPaginatedList, error) {
	args := m.Called(ctx, publicKey, pluginID, take, skip, activeFilter)
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

func (m *MockDatabaseStorage) MarkFeesCollected(ctx context.Context, dbTx pgx.Tx, ids []uint64, txHash string, totalAmount uint64) error {
	args := m.Called(ctx, dbTx, ids, txHash, totalAmount)
	return args.Error(0)
}

func (m *MockDatabaseStorage) GetFeeById(ctx context.Context, id uint64) (*types.Fee, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Fee), args.Error(1)
}

func (m *MockDatabaseStorage) GetFeesByPublicKey(ctx context.Context, publicKey string) ([]*types.Fee, error) {
	args := m.Called(ctx, publicKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*types.Fee), args.Error(1)
}

func (m *MockDatabaseStorage) GetFeesByPluginID(ctx context.Context, pluginID string, publicKey string, skip, take uint32) ([]itypes.FeeWithStatus, uint32, error) {
	args := m.Called(ctx, pluginID, publicKey, skip, take)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]itypes.FeeWithStatus), args.Get(1).(uint32), args.Error(2)
}

func (m *MockDatabaseStorage) GetPluginBillingSummary(ctx context.Context, publicKey string) ([]itypes.PluginBillingSummaryRow, error) {
	args := m.Called(ctx, publicKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]itypes.PluginBillingSummaryRow), args.Error(1)
}

func (m *MockDatabaseStorage) InsertFee(ctx context.Context, dbTx pgx.Tx, fee *types.Fee) (uint64, error) {
	args := m.Called(ctx, dbTx, fee)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockDatabaseStorage) InsertPluginInstallation(ctx context.Context, dbTx pgx.Tx, pluginID string, publicKey string) error {
	args := m.Called(ctx, dbTx, pluginID, publicKey)
	return args.Error(0)
}

func (m *MockDatabaseStorage) GetUserFees(ctx context.Context, publicKey string) (*types.UserFeeStatus, error) {
	args := m.Called(ctx, publicKey)
	return args.Get(0).(*types.UserFeeStatus), args.Error(1)
}

func (m *MockDatabaseStorage) UpdateBatchStatus(ctx context.Context, dbTx pgx.Tx, txHash string, status *rpc.TxOnChainStatus) error {
	args := m.Called(ctx, dbTx, txHash, status)
	return args.Error(0)
}

func (m *MockDatabaseStorage) IsTrialActive(ctx context.Context, dbTx pgx.Tx, pubKey string) (bool, time.Duration, error) {
	args := m.Called(ctx, dbTx, pubKey)
	return args.Get(0).(bool), args.Get(1).(time.Duration), args.Error(2)
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

func (m *MockDatabaseStorage) AttachTagToPlugin(ctx context.Context, pluginID string, tagID string) (*itypes.Plugin, error) {
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

func (m *MockDatabaseStorage) FindAvgRatingByPluginID(ctx context.Context, pluginID string) (itypes.PluginAvgRatingDto, error) {
	args := m.Called(ctx, pluginID)
	if args.Get(0) == nil {
		return itypes.PluginAvgRatingDto{}, args.Error(1)
	}
	return args.Get(0).(itypes.PluginAvgRatingDto), args.Error(1)
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

func (m *MockDatabaseStorage) GetAPIKeyByPluginId(ctx context.Context, pluginId string) (*itypes.APIKey, error) {
	args := m.Called(ctx, pluginId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.APIKey), args.Error(1)
}

func (m *MockDatabaseStorage) GetPluginTitlesByIDs(ctx context.Context, ids []string) (map[string]string, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockDatabaseStorage) GetPricingsByPluginIDs(ctx context.Context, pluginIDs []string) (map[string][]itypes.PricingInfo, error) {
	args := m.Called(ctx, pluginIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string][]itypes.PricingInfo), args.Error(1)
}

func (m *MockDatabaseStorage) UpsertReport(ctx context.Context, pluginID string, publicKey, reason, details string, cooldown time.Duration) error {
	args := m.Called(ctx, pluginID, publicKey, reason, details, cooldown)
	return args.Error(0)
}

func (m *MockDatabaseStorage) GetReport(ctx context.Context, pluginID string, publicKey string) (*itypes.PluginReport, error) {
	args := m.Called(ctx, pluginID, publicKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.PluginReport), args.Error(1)
}

func (m *MockDatabaseStorage) CountReportsInWindow(ctx context.Context, pluginID string, window time.Duration) (int, error) {
	args := m.Called(ctx, pluginID, window)
	return args.Int(0), args.Error(1)
}

func (m *MockDatabaseStorage) HasInstallation(ctx context.Context, pluginID string, publicKey string) (bool, error) {
	args := m.Called(ctx, pluginID, publicKey)
	return args.Bool(0), args.Error(1)
}

func (m *MockDatabaseStorage) CountInstallations(ctx context.Context, pluginID string) (int, error) {
	args := m.Called(ctx, pluginID)
	return args.Int(0), args.Error(1)
}

func (m *MockDatabaseStorage) IsPluginPaused(ctx context.Context, pluginID string) (bool, error) {
	args := m.Called(ctx, pluginID)
	return args.Bool(0), args.Error(1)
}

func (m *MockDatabaseStorage) PausePlugin(ctx context.Context, pluginID string, record itypes.PauseHistoryRecord) error {
	args := m.Called(ctx, pluginID, record)
	return args.Error(0)
}

func (m *MockDatabaseStorage) GetControlFlags(ctx context.Context, k1, k2 string) (map[string]bool, error) {
	args := m.Called(ctx, k1, k2)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]bool), args.Error(1)
}

func (m *MockDatabaseStorage) IsOwner(ctx context.Context, pluginID string, publicKey string) (bool, error) {
	args := m.Called(ctx, pluginID, publicKey)
	return args.Bool(0), args.Error(1)
}

func (m *MockDatabaseStorage) GetPluginsByOwner(ctx context.Context, publicKey string) ([]string, error) {
	args := m.Called(ctx, publicKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockDatabaseStorage) AddOwner(ctx context.Context, pluginID string, publicKey string, addedVia itypes.PluginOwnerAddedVia, addedBy string) error {
	args := m.Called(ctx, pluginID, publicKey, addedVia, addedBy)
	return args.Error(0)
}

func (m *MockDatabaseStorage) DeactivateOwner(ctx context.Context, pluginID string, publicKey string) error {
	args := m.Called(ctx, pluginID, publicKey)
	return args.Error(0)
}

func (m *MockDatabaseStorage) GetPluginImagesByPluginIDs(ctx context.Context, pluginIDs []string) ([]itypes.PluginImageRecord, error) {
	args := m.Called(ctx, pluginIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]itypes.PluginImageRecord), args.Error(1)
}

func (m *MockDatabaseStorage) GetPluginImageByType(ctx context.Context, pluginID string, imageType itypes.PluginImageType) (*itypes.PluginImageRecord, error) {
	args := m.Called(ctx, pluginID, imageType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.PluginImageRecord), args.Error(1)
}

func (m *MockDatabaseStorage) GetNextMediaOrder(ctx context.Context, pluginID string) (int, error) {
	args := m.Called(ctx, pluginID)
	return args.Int(0), args.Error(1)
}

func TestGenerateTokenPair(t *testing.T) {
	testCases := []struct {
		name          string
		secret        string
		publicKey     string
		expectedError bool
	}{
		{
			name:          "Valid token pair generation",
			secret:        "test-secret",
			publicKey:     testPublicKey,
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
			tokenPair, err := auth.GenerateTokenPair(context.Background(), tt.publicKey)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, tokenPair)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tokenPair)
				assert.NotEmpty(t, tokenPair.AccessToken)
				assert.NotEmpty(t, tokenPair.RefreshToken)
				assert.NotEqual(t, tokenPair.AccessToken, tokenPair.RefreshToken, "Access and refresh tokens should be different")
				assert.Equal(t, 3600, tokenPair.ExpiresIn, "Access token should expire in 3600 seconds (60 minutes)")

				accessClaims, err := auth.ValidateToken(context.Background(), tokenPair.AccessToken)
				assert.NoError(t, err)
				assert.NotNil(t, accessClaims)
				assert.Equal(t, service.TokenTypeAccess, accessClaims.TokenType)
				assert.Equal(t, tt.publicKey, accessClaims.PublicKey)

				mockDB.On("GetVaultToken", mock.Anything, mock.Anything).Return(&itypes.VaultToken{
					TokenID:   uuid.New().String(),
					PublicKey: tt.publicKey,
				}, nil)
				mockDB.On("UpdateVaultTokenLastUsed", mock.Anything, mock.Anything).Return(nil)

				refreshClaims, err := auth.ValidateToken(context.Background(), tokenPair.RefreshToken)
				assert.NoError(t, err)
				assert.NotNil(t, refreshClaims)
				assert.Equal(t, service.TokenTypeRefresh, refreshClaims.TokenType)
				assert.Equal(t, tt.publicKey, refreshClaims.PublicKey)
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
				tokenPair, _ := auth.GenerateTokenPair(context.Background(), "test-public-key")
				return tokenPair.RefreshToken
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
					PublicKey: "test-public-key",
					TokenID:   "test-token-id",
					TokenType: service.TokenTypeAccess,
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
					PublicKey: "test-public-key",
					TokenID:   "test-token-id",
					TokenType: service.TokenTypeAccess,
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
				tokenPair, _ := auth.GenerateTokenPair(context.Background(), "test-public-key")
				return tokenPair.RefreshToken
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
			name: "Valid refresh token",
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

				auth := service.NewAuthService(secret, mockDB, testLogger)
				tokenPair, _ := auth.GenerateTokenPair(context.Background(), testPublicKey)
				return tokenPair.RefreshToken
			},
			shouldError: false,
		},
		{
			name: "Expired refresh token",
			setupToken: func() string {
				claims := &service.Claims{
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
					},
					PublicKey: testPublicKey,
					TokenID:   "test-token-id",
					TokenType: service.TokenTypeRefresh,
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
		{
			name: "Access token used for refresh (should fail)",
			setupToken: func() string {
				claims := &service.Claims{
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
						IssuedAt:  jwt.NewNumericDate(time.Now()),
					},
					PublicKey: testPublicKey,
					TokenID:   uuid.New().String(),
					TokenType: service.TokenTypeAccess,
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString([]byte(secret))
				return tokenString
			},
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tokenString := tc.setupToken()
			mockDB := new(MockDatabaseStorage)

			if !tc.shouldError {
				mockDB.On("GetVaultToken", mock.Anything, mock.Anything).
					Return(&itypes.VaultToken{
						TokenID:   uuid.New().String(),
						PublicKey: testPublicKey,
					}, nil)
				mockDB.On("UpdateVaultTokenLastUsed", mock.Anything, mock.Anything).Return(nil)
			}

			authService := service.NewAuthService(secret, mockDB, testLogger)
			tokenPair, err := authService.RefreshToken(context.Background(), tokenString)

			if tc.shouldError {
				assert.Error(t, err)
				assert.Nil(t, tokenPair)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tokenPair)
				assert.NotEmpty(t, tokenPair.AccessToken)
				assert.NotEmpty(t, tokenPair.RefreshToken)
				assert.Equal(t, tokenString, tokenPair.RefreshToken, "Refresh token should remain the same")
				assert.NotEqual(t, tokenString, tokenPair.AccessToken, "Access token should be different from refresh token")
				assert.Equal(t, 3600, tokenPair.ExpiresIn, "Access token should expire in 3600 seconds (60 minutes)")

				claims, validationErr := authService.ValidateToken(context.Background(), tokenPair.AccessToken)
				assert.NoError(t, validationErr)
				assert.NotNil(t, claims)
				assert.Equal(t, service.TokenTypeAccess, claims.TokenType, "New token should be an access token")
			}
		})
	}
}
