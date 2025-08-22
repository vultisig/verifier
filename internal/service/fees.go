package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	abi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/jackc/pgx/v5"
	reth "github.com/vultisig/recipes/ethereum"

	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	resolver "github.com/vultisig/recipes/resolver"
	rtypes "github.com/vultisig/recipes/types"

	ecommon "github.com/ethereum/go-ethereum/common"
	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/types"
	ptypes "github.com/vultisig/verifier/types"
)

type Fees interface {
	PublicKeyGetFeeInfo(ctx context.Context, publicKey string, since *time.Time) ([]ptypes.FeeDebit, error)
	GetFeeBalance(ctx context.Context, publicKey string) (int64, error)
	GetFeeBalanceUnlocked(ctx context.Context, publicKey string) (int64, error)
}

var _ Fees = (*FeeService)(nil)

type FeeService struct {
	db        storage.DatabaseStorage
	logger    *logrus.Logger
	client    *asynq.Client
	feeConfig config.FeesConfig
	//TODO explore a per policy mutex to lock by policy ID without a map growing too big.
	SignRequestMutex sync.Mutex // This mutex is locked when a sign request is sent and unlocked after it is processed. This is because a fee_credit is created on a sign request. It is to avoid race conditions of 2 credits being created at the same time.
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

// This function returns a list of all fees incurred for an "account"/"public key"
func (s *FeeService) PublicKeyGetFeeInfo(ctx context.Context, publicKey string, since *time.Time) ([]ptypes.FeeDebit, error) {
	fees, err := s.db.GetFeeDebitsByPublicKey(ctx, publicKey, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get fees: %w", err)
	}
	return fees, nil
}

// Sum of all fee_debits and credits for an "account"/"public key"
func (s *FeeService) GetFeeBalance(ctx context.Context, publicKey string) (int64, error) {
	s.SignRequestMutex.Lock()
	defer s.SignRequestMutex.Unlock()
	return s.GetFeeBalanceUnlocked(ctx, publicKey)
}

// GetFeeBalanceUnlocked is the internal version that doesn't acquire the mutex
// This should only be called when the SignRequestMutex is already held
func (s *FeeService) GetFeeBalanceUnlocked(ctx context.Context, publicKey string) (int64, error) {
	feesOwed, err := s.db.GetFeesOwed(ctx, publicKey)
	if err != nil {
		return 0, fmt.Errorf("failed to get fees: %w", err)
	}
	return feesOwed, nil
}

func (s *FeeService) CreateFeeCredit(ctx context.Context, tx *pgx.Tx, feeCredit types.FeeCredit) error {
	var err error
	if tx == nil {
		*tx, err = s.db.Pool().Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer func() {
			if err != nil {
				(*tx).Rollback(ctx)
				return
			}
			(*tx).Commit(ctx)
		}()
	}

	_, err = s.db.InsertFeeCreditTx(ctx, *tx, feeCredit)
	if err != nil {
		return fmt.Errorf("failed to insert fee credit: %w", err)
	}
	return nil
}

func (s *FeeService) CreateFeeDebit(ctx context.Context, tx *pgx.Tx, feeDebit types.FeeDebit) error {
	var err error
	if tx == nil {
		*tx, err = s.db.Pool().Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer func() {
			if err != nil {
				(*tx).Rollback(ctx)
				return
			}
			(*tx).Commit(ctx)
		}()
	}

	_, err = s.db.InsertFeeDebitTx(ctx, *tx, feeDebit)
	if err != nil {
		return fmt.Errorf("failed to insert fee debit: %w", err)
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

// parseTransactionData decodes and validates the transaction data from a raw message
func (s *FeeService) parseTransactionData(rawMessage string) (*unsignedDynamicFeeTx, error) {
	tx, err := decodeTx(rawMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to decode fee tx: %w", err)
	}

	// Get the input data (calldata)
	data := tx.Data
	if len(data) < 4 {
		return nil, fmt.Errorf("no data found in tx")
	}

	// Check for contract creation (tx.To == nil)
	if tx.To == nil {
		return nil, fmt.Errorf("contract creation transactions are not supported for fee validation")
	}

	contractAddress := tx.To.Hex()
	if contractAddress != s.feeConfig.USDCAddress {
		return nil, fmt.Errorf("transaction must be sent to the configured usdc contract address")
	}

	return tx, nil
}

// parseERC20TransferArgs extracts the recipient and amount from ERC20 transfer calldata
func (s *FeeService) parseERC20TransferArgs(data []byte) (recipient ecommon.Address, amount *big.Int, err error) {
	// Parse the ERC20 transfer ABI
	const transferABI = `[{"name":"transfer","type":"function","inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]}]`
	parsedABI, err := abi.JSON(strings.NewReader(transferABI))
	if err != nil {
		return ecommon.Address{}, nil, fmt.Errorf("failed to parse ABI")
	}

	// Get the method by selector
	method, err := parsedABI.MethodById(data[:4])
	if err != nil {
		return ecommon.Address{}, nil, fmt.Errorf("unknown method ID")
	}

	// Decode the arguments
	args := make(map[string]interface{})
	if err := method.Inputs.UnpackIntoMap(args, data[4:]); err != nil {
		return ecommon.Address{}, nil, fmt.Errorf("failed get recipient and amount from tx")
	}

	recipient, ok := args["to"].(ecommon.Address)
	if !ok {
		return ecommon.Address{}, nil, fmt.Errorf("invalid usdc address")
	}

	amount, ok = args["value"].(*big.Int)
	if !ok {
		return ecommon.Address{}, nil, fmt.Errorf("invalid amount")
	}

	return recipient, amount, nil
}

// validateTreasuryRecipient checks if the recipient matches the treasury address
func (s *FeeService) validateTreasuryRecipient(recipient ecommon.Address) error {
	treasuryResolver := resolver.NewDefaultTreasuryResolver()
	treasuryConstant := rtypes.MagicConstant_VULTISIG_TREASURY
	treasuryRecipientString, _, err := treasuryResolver.Resolve(treasuryConstant, "ethereum", "usdc")
	if err != nil {
		return fmt.Errorf("failed to resolve treasury")
	}

	treasuryRecipient := ecommon.HexToAddress(treasuryRecipientString)
	if recipient.Cmp(treasuryRecipient) != 0 {
		return fmt.Errorf("recipient is not the treasury")
	}

	return nil
}

// validateTransactionHash validates that the transaction hash matches the expected hash
func (s *FeeService) validateTransactionHash(req *ptypes.PluginKeysignRequest) error {
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

	return nil
}

func (s *FeeService) ValidateFees(ctx context.Context, req *ptypes.PluginKeysignRequest) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	if len(req.Messages) != 1 {
		return fmt.Errorf("only one tx per fee run is supported")
	}

	// Validate transaction hash
	if err := s.validateTransactionHash(req); err != nil {
		return err
	}

	// Parse transaction data
	tx, err := s.parseTransactionData(req.Messages[0].RawMessage)
	if err != nil {
		return err
	}

	// Parse ERC20 transfer arguments
	recipient, amount, err := s.parseERC20TransferArgs(tx.Data)
	if err != nil {
		return err
	}

	// Validate treasury recipient
	if err := s.validateTreasuryRecipient(recipient); err != nil {
		return err
	}

	// Validate fee amount against what's due
	feesDueInt64, err := s.GetFeeBalanceUnlocked(ctx, req.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to get fee balance: %w", err)
	}

	//TODO could add a nominal amount here to prevent collecting tiny fees that would cost more than they're worth in gas
	if feesDueInt64 <= 0 {
		return fmt.Errorf("fees negative or zero")
	}

	feesDue := big.NewInt(0).SetInt64(feesDueInt64)

	if amount.Cmp(feesDue) > 0 {
		return fmt.Errorf("fee amount exceeds fees due")
	}

	return nil
}

func (s *FeeService) GetAmountFromSignRequest(ctx context.Context, req *ptypes.PluginKeysignRequest) (uint64, error) {
	if req == nil {
		return 0, fmt.Errorf("request is nil")
	}

	if len(req.Messages) != 1 {
		return 0, fmt.Errorf("only one tx per fee run is supported")
	}

	// Parse transaction data
	tx, err := s.parseTransactionData(req.Messages[0].RawMessage)
	if err != nil {
		return 0, fmt.Errorf("failed to parse transaction data: %w", err)
	}

	// Parse ERC20 transfer arguments to extract the amount
	_, amount, err := s.parseERC20TransferArgs(tx.Data)
	if err != nil {
		return 0, fmt.Errorf("failed to parse ERC20 transfer arguments: %w", err)
	}

	// Convert big.Int to uint64
	if !amount.IsUint64() {
		return 0, fmt.Errorf("amount too large to fit in uint64")
	}

	return amount.Uint64(), nil
}
