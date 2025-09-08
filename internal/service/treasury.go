package service

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/types"
	ptypes "github.com/vultisig/verifier/types"
)

type Treasury interface {
	CreateTreasuryLedgerRecord(ctx context.Context, batch ptypes.FeeBatch) error
}

var _ Treasury = (*TreasuryService)(nil)

type TreasuryService struct {
	db     storage.DatabaseStorage
	logger *logrus.Logger
	client *asynq.Client
	//TODO explore a per policy mutex to lock by policy ID without a map growing too big.
	SignRequestMutex sync.Mutex // This mutex is locked when a sign request is sent and unlocked after it is processed. This is because a fee_credit is created on a sign request. It is to avoid race conditions of 2 credits being created at the same time.
}

func NewTreasuryService(db storage.DatabaseStorage, client *asynq.Client, logger *logrus.Logger) (*TreasuryService, error) {
	return &TreasuryService{
		db:               db,
		logger:           logger,
		client:           client,
		SignRequestMutex: sync.Mutex{},
	}, nil
}

type developerProfile struct {
	Developer types.Developer
	Cut       *big.Float
}

// Perform some logic to find out the exact split for a developer and internal treasury.
func (ts *TreasuryService) getDeveloperSplit(ctx context.Context, feeBatch ptypes.FeeBatch) (developerProfile, error) {
	return developerProfile{
		Developer: types.Developer{
			ID: uuid.Nil,
		},
		Cut: big.NewFloat(0.7),
	}, nil
}

// only one debit record can be added to the treasury ledger for a fee batch collection. So that check is enforced at DB level.
func (ts *TreasuryService) CreateTreasuryLedgerRecord(ctx context.Context, batch ptypes.FeeBatch) error {

	_batch, err := ts.db.GetFeeBatch(ctx, batch.ID)
	if err != nil {
		return fmt.Errorf("failed to get fee batch: %w", err)
	}

	if _batch.Status != ptypes.FeeBatchStatusCompleted {
		return fmt.Errorf("fee batch is not completed")
	}

	developerSplit, err := ts.getDeveloperSplit(ctx, *_batch)
	if err != nil {
		return fmt.Errorf("failed to get treasury split: %w", err)
	}

	batchAmount, err := ts.db.GetFeeBatchAmount(ctx, _batch.ID)
	if err != nil {
		return fmt.Errorf("failed to get fee batch amount: %w", err)
	}

	developerAmount := big.NewFloat(0).SetUint64(batchAmount)
	developerAmount = developerAmount.Mul(developerAmount, developerSplit.Cut)
	developerAmountUint64, _ := developerAmount.Uint64()
	treasuryAmountUint64 := batchAmount - developerAmountUint64

	tx, err := ts.db.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	var rollbackError error
	defer func() {
		if rollbackError != nil {
			tx.Rollback(ctx)
		}
	}()

	_, rollbackError = ts.db.CreateTreasuryLedgerDebitFromFeeBatch(ctx, tx, _batch.ID, developerAmountUint64, developerSplit.Developer.ID, "")
	if rollbackError != nil {

		return fmt.Errorf("failed to create treasury account record: %w", rollbackError)
	}

	_, rollbackError = ts.db.CreateVultisigDuesDebitFromFeeBatch(ctx, tx, _batch.ID, treasuryAmountUint64)
	if rollbackError != nil {
		return fmt.Errorf("failed to create developer account record: %w", rollbackError)
	}

	return tx.Commit(ctx)
}
