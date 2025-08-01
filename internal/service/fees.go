package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	abi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/jackc/pgx/v5"
	reth "github.com/vultisig/recipes/ethereum"

	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	resolver "github.com/vultisig/recipes/resolver"
	rtypes "github.com/vultisig/recipes/types"

	ecommon "github.com/ethereum/go-ethereum/common"
	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/storage"
	itypes "github.com/vultisig/verifier/internal/types"
	ptypes "github.com/vultisig/verifier/types"
)

type Fees interface {
	PublicKeyGetFeeInfo(ctx context.Context, publicKey string) (*itypes.FeeHistoryDto, error)
	MarkFeesCollected(ctx context.Context, collectedAt time.Time, ids []uuid.UUID, txHash string) ([]uuid.UUID, error)
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

func (s *FeeService) PublicKeyGetFeeInfo(ctx context.Context, publicKey string) (*itypes.FeeHistoryDto, error) {

	fees, err := s.db.GetFeesByPublicKey(ctx, publicKey, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get fees: %w", err)
	}

	var totalFeesIncurred uint64
	var feesPendingCollection uint64

	ifees := make([]itypes.FeeDto, 0, len(fees))
	for _, fee := range fees {
		collected := true
		if fee.CollectedAt == nil {
			collected = false
		}
		collectedDt := ""
		if collected {
			collectedDt = fee.CollectedAt.Format(time.RFC3339)
		}
		ifee := itypes.FeeDto{
			ID:          fee.ID,
			PublicKey:   fee.PublicKey,
			PolicyId:    fee.PolicyID,
			PluginId:    fee.PluginID.String(),
			Amount:      fee.Amount,
			Collected:   collected,
			CollectedAt: collectedDt,
			ChargedAt:   fee.ChargedAt.Format(time.RFC3339),
		}
		totalFeesIncurred += fee.Amount
		if !collected {
			feesPendingCollection += fee.Amount
		}
		ifees = append(ifees, ifee)
	}

	return &itypes.FeeHistoryDto{
		Fees:                  ifees,
		TotalFeesIncurred:     totalFeesIncurred,
		FeesPendingCollection: feesPendingCollection,
	}, nil
}

func (s *FeeService) MarkFeesCollected(ctx context.Context, collectedAt time.Time, ids []uuid.UUID, txHash string) ([]uuid.UUID, error) {
	var err error
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	fees, err := s.db.GetFees(ctx, ids...)
	if err != nil {
		return nil, fmt.Errorf("failed to get fees: %w", err)
	}

	for _, fee := range fees {
		if fee.CollectedAt != nil {
			return nil, fmt.Errorf("fee already collected")
		}
	}

	err = s.db.MarkFeesCollected(ctx, tx, collectedAt, ids, txHash)
	if err != nil {
		return nil, fmt.Errorf("db failed to mark fees as collected: %w", err)
	}

	for _, id := range ids {
		fees, err := s.db.GetFees(ctx, id)
		if err != nil || len(fees) != 1 {
			return nil, fmt.Errorf("failed to get fee: %w", err)
		}

		fee := fees[0]
		fee.CollectedAt = &collectedAt
		err = s.createTreasuryLedgerRecord(ctx, tx, fee)
		if err != nil {
			return nil, fmt.Errorf("failed to create treasury ledger record: %w", err)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to commit tx: %w", err)
	}
	return ids, nil
}

// returns the amount that is kept by treasury, the amount sent to the developer is 1 - this value
func (s *FeeService) getTreasurySplit(ctx context.Context, fee ptypes.Fee) (float64, uuid.UUID, error) {
	pluginId := fee.PluginID
	pluginId = pluginId
	//TODO additional work can be done here to get the treasury split, for now it is hardcoded

	return 0.3, uuid.Nil, nil
}

func (s *FeeService) createTreasuryLedgerRecord(ctx context.Context, tx pgx.Tx, fee ptypes.Fee) error {
	if fee.CollectedAt == nil {
		return fmt.Errorf("fee is not collected")
	}

	treasurySplit, developerId, err := s.getTreasurySplit(ctx, fee)
	if err != nil {
		return fmt.Errorf("failed to get treasury split: %w", err)
	}

	treasuryAmount := big.NewFloat(0).SetUint64(fee.Amount)
	treasuryAmount = treasuryAmount.Mul(treasuryAmount, big.NewFloat(treasurySplit))
	treasuryAmountUint64, _ := treasuryAmount.Uint64()
	developerAmountUint64 := fee.Amount - treasuryAmountUint64

	treasuryAccount := ptypes.TreasuryLedgerRecord{
		Type:   ptypes.TreasuryLedgerEntryTypeFeeCredit,
		Amount: treasuryAmountUint64,
		FeeID:  &fee.ID,
	}

	developerAccount := ptypes.TreasuryLedgerRecord{
		Type:        ptypes.TreasuryLedgerEntryTypeFeeCredit,
		Amount:      developerAmountUint64,
		FeeID:       &fee.ID,
		DeveloperID: &developerId,
	}

	err = s.db.CreateTreasuryLedgerRecord(ctx, tx, treasuryAccount)
	if err != nil {
		return fmt.Errorf("failed to create treasury account record: %w", err)
	}

	err = s.db.CreateTreasuryLedgerRecord(ctx, tx, developerAccount)
	if err != nil {
		return fmt.Errorf("failed to create developer account record: %w", err)
	}
	return nil
}

type unsignedDynamicFeeTx struct {
	ChainID    *big.Int
	Nonce      uint64
	GasTipCap  *big.Int
	GasFeeCap  *big.Int
	Gas        uint64
	To         *ecommon.Address
	Value      *big.Int
	Data       []byte
	AccessList etypes.AccessList
}

func decodeTx(rawHex string) (*unsignedDynamicFeeTx, error) {
	rawHex = strings.TrimPrefix(rawHex, "0x")
	rawBytes, err := hex.DecodeString(rawHex)

	if err != nil {
		return nil, fmt.Errorf("hex decode failed: %w", err)
	}

	// Check transaction type (EIP-1559 is 0x02)
	if len(rawBytes) == 0 || rawBytes[0] != 0x02 {
		return nil, fmt.Errorf("unsupported transaction type: 0x%02x", rawBytes[0])
	}

	tx := new(unsignedDynamicFeeTx)
	err = rlp.DecodeBytes(rawBytes[1:], tx)
	if err != nil {
		return nil, fmt.Errorf("rlp decode failed: %w", err)
	}

	return tx, nil
}

func (s *FeeService) ValidateFees(ctx context.Context, req *ptypes.PluginKeysignRequest) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	if len(req.Messages) != 1 {
		return fmt.Errorf("only one tx per fee run is supported")
	}

	b64DecodedMessage, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.Messages[0].Message))
	if err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}

	txRawMessage := req.Messages[0].RawMessage
	txData, err := hexutil.Decode(txRawMessage)
	if err != nil {
		return fmt.Errorf("failed to decode tx data: %w", err)
	}

	decodedTxData, err := reth.DecodeUnsignedPayload(txData)
	if err != nil {
		return fmt.Errorf("cannot decode raw tx: %w", err)
	}

	txHashToSign := etypes.LatestSignerForChainID(big.NewInt(1)).Hash(etypes.NewTx(decodedTxData))
	if !bytes.Equal(txHashToSign.Bytes(), b64DecodedMessage) {
		return fmt.Errorf("tx hash mismatch")
	}

	// Unmarshal the transaction (handles EIP-1559, legacy, etc.)
	tx, err := decodeTx((*req).Messages[0].RawMessage)
	if err != nil {
		return fmt.Errorf("failed to decode fee tx: %w", err)
	}

	// Get the input data (calldata)
	data := tx.Data
	if len(data) < 4 {
		return fmt.Errorf("no data found in tx")
	}

	// Check for contract creation (tx.To == nil)
	if tx.To == nil {
		return fmt.Errorf("contract creation transactions are not supported for fee validation")
	}

	contractAddress := tx.To.Hex()
	if contractAddress != s.feeConfig.USDCAddress {
		return fmt.Errorf("transaction must be sent to the configured usdc contract address")
	}

	// Parse the ERC20 transfer ABI
	const transferABI = `[{"name":"transfer","type":"function","inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]}]`
	parsedABI, err := abi.JSON(strings.NewReader(transferABI))
	if err != nil {
		return fmt.Errorf("failed to parse ABI")
	}

	// Get the method by selector
	method, err := parsedABI.MethodById(data[:4])
	if err != nil {
		return fmt.Errorf("unknown method ID")
	}

	// Decode the arguments
	args := make(map[string]interface{})
	if err := method.Inputs.UnpackIntoMap(args, data[4:]); err != nil {
		return fmt.Errorf("failed get recipient and amount from tx")
	}

	recipient, ok := args["to"].(ecommon.Address)
	if !ok {
		return fmt.Errorf("invalid usdc address")
	}

	treasuryResolver := resolver.NewDefaultTreasuryResolver()
	treasuryConstant := rtypes.MagicConstant_VULTISIG_TREASURY
	treasuryRecipientString, _, err := treasuryResolver.Resolve(treasuryConstant, "ethereum", "usdc")
	treasuryRecipient := ecommon.HexToAddress(treasuryRecipientString)
	if err != nil {
		return fmt.Errorf("failed to resolve treasury")
	}

	if recipient.Cmp(treasuryRecipient) != 0 {
		return fmt.Errorf("recipient is not the treasury")
	}

	// Check valid usdc address
	if strings.TrimPrefix(strings.ToLower(s.feeConfig.USDCAddress), "0x") != strings.TrimPrefix(strings.ToLower(contractAddress), "0x") {
		return fmt.Errorf("transaction must be sent to the configured usdc contract address")
	}

	customFields := req.Messages[0].CustomFields
	if customFields == nil {
		return fmt.Errorf("custom fields are required")
	}

	fidsI, ok := customFields["fee_ids"].([]interface{})
	if !ok {
		return fmt.Errorf("fee ids are required")
	}

	fids := make([]uuid.UUID, 0, len(fidsI))

	for _, fidI := range fidsI {
		fid, ok := fidI.(string)
		if !ok {
			return fmt.Errorf("fee ids are not a list of uuid")
		}
		id, err := uuid.Parse(fid)
		if err != nil {
			return fmt.Errorf("fee ids are not a list of uuid")
		}
		fids = append(fids, id)
	}

	fees, err := s.db.GetFees(ctx, fids...)
	if err != nil {
		return fmt.Errorf("failed to get fees: %w", err)
	}

	if len(fees) != len(fids) {
		return fmt.Errorf("fee ids are not valid")
	}

	amountRequested := big.NewInt(0)
	for _, fee := range fees {
		if fee.CollectedAt != nil {
			return fmt.Errorf("fee already collected")
		}
		amountRequested = amountRequested.Add(amountRequested, big.NewInt(0).SetUint64(fee.Amount))
	}

	amount, ok := args["value"].(*big.Int)
	if !ok {
		return fmt.Errorf("invalid amount")
	}

	if amountRequested.Cmp(amount) != 0 {
		return fmt.Errorf("fee amount incorrect")
	}

	return nil
}
