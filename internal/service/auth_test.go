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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDatabaseStorage is a mock implementation of storage.DatabaseStorage
type MockDatabaseStorage struct {
	mock.Mock
}

// noopTx is a lightweight transaction stub that satisfies pgx.Tx interface
type noopTx struct{}

func (t *noopTx) Begin(_ context.Context) (pgx.Tx, error) { return t, nil }
func (t *noopTx) Commit(_ context.Context) error          { return nil }
func (t *noopTx) Rollback(_ context.Context) error        { return nil }
func (t *noopTx) Exec(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (t *noopTx) Query(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
	return nil, nil
}
func (t *noopTx) QueryRow(_ context.Context, _ string, _ ...interface{}) pgx.Row {
	return nil
}
func (t *noopTx) Conn() *pgx.Conn {
	return nil
}
func (t *noopTx) CopyFrom(_ context.Context, _ pgx.Identifier, _ []string, _ pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (t *noopTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}
func (t *noopTx) Prepare(_ context.Context, _ string, _ string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (t *noopTx) SendBatch(_ context.Context, _ *pgx.Batch) pgx.BatchResults {
	return nil
}

// mockPool is a lightweight pool stub that satisfies pgxpool.Pool interface
type mockPool struct {
	pgxpool.Pool
}

func (m *mockPool) Begin(ctx context.Context) (pgx.Tx, error) {
	return &noopTx{}, nil
}

func (m *mockPool) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return &noopTx{}, nil
}

func (m *mockPool) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	return nil, nil
}

func (m *mockPool) AcquireAllIdle(ctx context.Context) []*pgxpool.Conn {
	return nil
}

func (m *mockPool) AcquireFunc(ctx context.Context, f func(*pgxpool.Conn) error) error {
	return nil
}

func (m *mockPool) BeginFunc(ctx context.Context, f func(pgx.Tx) error) error {
	return nil
}

func (m *mockPool) BeginTxFunc(ctx context.Context, txOptions pgx.TxOptions, f func(pgx.Tx) error) error {
	return nil
}

func (m *mockPool) Close() {}

func (m *mockPool) Config() *pgxpool.Config {
	return nil
}

func (m *mockPool) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (m *mockPool) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (m *mockPool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return nil
}

func (m *mockPool) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}

func (m *mockPool) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (m *mockPool) Ping(ctx context.Context) error {
	return nil
}

func (m *mockPool) Stat() *pgxpool.Stat {
	return nil
}

func (m *MockDatabaseStorage) Pool() *pgxpool.Pool {
	return &pgxpool.Pool{}
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
	return args.Get(0).(int64), args.Error(1)
}

// CreatePlugin is a stub to satisfy the DatabaseStorage interface
func (m *MockDatabaseStorage) CreatePlugin(ctx context.Context, pluginDto itypes.PluginCreateDto) (*itypes.Plugin, error) {
	args := m.Called(ctx, pluginDto)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.Plugin), args.Error(1)
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
	return args.Get(0).(uuid.UUID), args.Error(1)
}

// CreateTransactionHistoryTx is a stub to satisfy the DatabaseStorage interface
func (m *MockDatabaseStorage) CreateTransactionHistoryTx(ctx context.Context, dbTx pgx.Tx, tx itypes.TransactionHistory) (uuid.UUID, error) {
	args := m.Called(ctx, dbTx, tx)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockDatabaseStorage) DeletePluginById(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDatabaseStorage) FindPluginById(ctx context.Context, id uuid.UUID) (*itypes.Plugin, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.Plugin), args.Error(1)
}

func (m *MockDatabaseStorage) FindPlugins(ctx context.Context, take int, skip int, sort string) (itypes.PlugisDto, error) {
	args := m.Called(ctx, take, skip, sort)
	return args.Get(0).(itypes.PlugisDto), args.Error(1)
}

func (m *MockDatabaseStorage) UpdatePlugin(ctx context.Context, id uuid.UUID, updates itypes.PluginUpdateDto) (*itypes.Plugin, error) {
	args := m.Called(ctx, id, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.Plugin), args.Error(1)
}

func (m *MockDatabaseStorage) GetPluginPolicy(ctx context.Context, id uuid.UUID) (types.PluginPolicy, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(types.PluginPolicy), args.Error(1)
}

func (m *MockDatabaseStorage) GetAllPluginPolicies(ctx context.Context, publicKey string, pluginType string) ([]types.PluginPolicy, error) {
	args := m.Called(ctx, publicKey, pluginType)
	return args.Get(0).([]types.PluginPolicy), args.Error(1)
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

func (m *MockDatabaseStorage) FindPricingById(ctx context.Context, id string) (*itypes.Pricing, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*itypes.Pricing), args.Error(1)
}

func (m *MockDatabaseStorage) DeletePricingById(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDatabaseStorage) UpdateTransactionStatusTx(ctx context.Context, dbTx pgx.Tx, txID uuid.UUID, status itypes.TransactionStatus, metadata map[string]interface{}) error {
	args := m.Called(ctx, dbTx, txID, status, metadata)
	return args.Error(0)
}

func (m *MockDatabaseStorage) GetTransactionHistory(ctx context.Context, policyID uuid.UUID, transactionType string, take int, skip int) ([]itypes.TransactionHistory, error) {
	args := m.Called(ctx, policyID, transactionType, take, skip)
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

func TestGenerateToken(t *testing.T) {
	tests := []struct {
		name          string
		secret        string
		expectedError bool
	}{
		{
			name:          "Valid secret",
			secret:        "test-secret",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDatabaseStorage)
			authService := service.NewAuthService(tt.secret, mockDB)

			// Setup mock expectations
			mockDB.On("CreateVaultToken", mock.Anything, mock.Anything).Return(&itypes.VaultToken{
				ID:        uuid.New().String(),
				PublicKey: "test-public-key",
			}, nil)

			token, err := authService.GenerateToken("test-public-key")
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
	testPublicKey := "0x1234567890abcdef"

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

				auth := service.NewAuthService(secret, mockDB)
				token, _ := auth.GenerateToken(testPublicKey)
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

				auth := service.NewAuthService(secret, mockDB)
				token, _ := auth.GenerateToken(testPublicKey)
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

			authService := service.NewAuthService(tc.secret, mockDB)

			claims, err := authService.ValidateToken(tokenString)

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
	testPublicKey := "0x1234567890abcdef"

	testCases := []struct {
		name        string
		setupToken  func() string
		shouldError bool
	}{
		{
			name: "Valid token refresh",
			setupToken: func() string {
				mockDB := new(MockDatabaseStorage)
				mockDB.On("CreateVaultToken", mock.Anything, mock.Anything).Return(nil, nil)
				mockDB.On("GetVaultToken", mock.Anything, mock.Anything).Return(nil, nil)
				mockDB.On("UpdateVaultTokenLastUsed", mock.Anything, mock.Anything).Return(nil)
				mockDB.On("RevokeVaultToken", mock.Anything, mock.Anything).Return(nil)

				auth := service.NewAuthService(secret, mockDB)
				token, _ := auth.GenerateToken(testPublicKey)
				time.Sleep(1 * time.Second)
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
			mockDB.On("CreateVaultToken", mock.Anything, mock.Anything).Return(nil, nil)
			mockDB.On("GetVaultToken", mock.Anything, mock.Anything).Return(nil, nil)
			mockDB.On("UpdateVaultTokenLastUsed", mock.Anything, mock.Anything).Return(nil)
			mockDB.On("RevokeVaultToken", mock.Anything, mock.Anything).Return(nil)

			authService := service.NewAuthService(secret, mockDB)

			// For valid tokens, we need to guarantee a different ExpiresAt
			if !tc.shouldError {
				time.Sleep(1 * time.Second)
			}

			newToken, err := authService.RefreshToken(tokenString)

			if tc.shouldError {
				assert.Error(t, err)
				assert.Empty(t, newToken)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, newToken)
				assert.NotEqual(t, tokenString, newToken, "Refreshed token should be different from the original")

				claims, validationErr := authService.ValidateToken(newToken)
				assert.NoError(t, validationErr)
				assert.NotNil(t, claims)
			}
		})
	}
}
