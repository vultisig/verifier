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

	return nil
}
