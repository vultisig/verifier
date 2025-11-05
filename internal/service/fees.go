package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	abi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	reth "github.com/vultisig/recipes/ethereum"
	"github.com/vultisig/vultisig-go/common"

	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	resolver "github.com/vultisig/recipes/resolver"
	rtypes "github.com/vultisig/recipes/types"

	ecommon "github.com/ethereum/go-ethereum/common"
	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/storage"
	ptypes "github.com/vultisig/verifier/types"
)

type Fees interface {
	PublicKeyGetFeeInfo(ctx context.Context, publicKey string) ([]*ptypes.Fee, error)
	ValidateFees(ctx context.Context, req *ptypes.PluginKeysignRequest) error
	MarkFeesCollected(ctx context.Context, id uint64, txHash, network string, amount uint64) error
}

var _ Fees = (*FeeService)(nil)

type FeeService struct {
	db        storage.DatabaseStorage
	logger    *logrus.Logger
	client    *asynq.Client
	feeConfig config.FeesConfig
}

func NewFeeService(db storage.DatabaseStorage,
	client *asynq.Client, logger *logrus.Logger, feeConfig config.FeesConfig) (*FeeService, error) {
	if db == nil {
		return nil, fmt.Errorf("database storage cannot be nil")
	}
	return &FeeService{
		db:        db,
		logger:    logger.WithField("service", "fee").Logger,
		client:    client,
		feeConfig: feeConfig,
	}, nil
}

func (s *FeeService) PublicKeyGetFeeInfo(ctx context.Context, publicKey string) ([]*ptypes.Fee, error) {
	return s.db.GetFeesByPublicKey(ctx, publicKey)
}

func (s *FeeService) MarkFeesCollected(ctx context.Context, id uint64, txHash, network string, amount uint64) error {
	chain, err := common.FromString(network)
	if err != nil {
		return err
	}

	metadata := ptypes.CreditMetadata{
		DebitFeeID: id,
		TxHash:     txHash,
		Network:    chain.String(),
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed marshaling metadata: %w", err)
	}

	feeInfo, err := s.db.GetFeeById(ctx, id)
	if err != nil {
		return fmt.Errorf("failed fetching fee: %w", err)
	}

	creditFee := &ptypes.Fee{
		PolicyID:       feeInfo.PolicyID,
		PublicKey:      feeInfo.PublicKey,
		TxType:         ptypes.TxTypeCredit,
		Amount:         amount,
		CreatedAt:      time.Now(),
		FeeType:        "blockchain_fee",
		Metadata:       metadataJSON,
		UnderlyingType: "refund",
		UnderlyingID:   txHash,
	}
	err = s.db.InsertFee(ctx, nil, creditFee)
	if err != nil {
		return fmt.Errorf("failed inserting fee: %w", err)
	}
	return nil
}
