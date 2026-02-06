package tx_indexer

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/config"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/conv"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/rpc"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/storage"
	"github.com/vultisig/vultisig-go/common"
)

func createWorker() (*Worker, context.CancelFunc, storage.TxIndexerRepo, error) {
	ctx, stop := context.WithCancel(context.Background())

	logger := logrus.New()

	var cfg config.Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, stop, nil, fmt.Errorf("failed to load config: %w", err)
	}

	db, err := storage.NewPostgresTxIndexStore(ctx, cfg.Database.DSN)
	if err != nil {
		return nil, stop, nil, fmt.Errorf("postgres.NewPostgresBackend: %w", err)
	}

	rpcs, err := Rpcs(ctx, cfg.Rpc)
	if err != nil {
		return nil, stop, nil, fmt.Errorf("rpc: %w", err)
	}

	worker := NewWorker(
		logger,
		cfg.Interval,
		cfg.IterationTimeout,
		cfg.MarkLostAfter,
		cfg.Concurrency,
		db,
		rpcs,
		nil, // metrics disabled for tests
	)

	return worker, stop, db, nil
}

func TestWorker_getMarkLostAfter(t *testing.T) {
	worker := &Worker{
		markLostAfter: 3 * time.Hour, // default for UTXO
	}

	tests := []struct {
		chain    common.Chain
		expected time.Duration
	}{
		{common.Solana, 2 * time.Minute},
		{common.XRP, 5 * time.Minute},
		{common.Ethereum, 30 * time.Minute},
		{common.Base, 30 * time.Minute},
		{common.Arbitrum, 30 * time.Minute},
		{common.Bitcoin, 3 * time.Hour},
		{common.Litecoin, 3 * time.Hour},
		{common.Dogecoin, 3 * time.Hour},
	}

	for _, tc := range tests {
		t.Run(tc.chain.String(), func(t *testing.T) {
			got := worker.getMarkLostAfter(tc.chain)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestWorker_positive(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		return
	}

	ctx := context.Background()

	worker, stop, db, createErr := createWorker()
	require.Nil(t, createErr)
	defer stop()

	type suite struct {
		chain         common.Chain
		hash          string
		fromPublicKey string
		toPublicKey   string
	}
	for _, testcase := range []suite{{
		chain:         common.Bitcoin,
		hash:          "75149b57b808eb2083bada72018c48118238427a229e75896e30ccd09646df8e",
		fromPublicKey: "1Hk4sk6kNscQ32ZZ9eGMtM2URNbYYRXLtX",
		toPublicKey:   "195pYUenhG2FbedE2mYDptWRHZESMbreie",
	}, {
		chain:         common.Ethereum,
		hash:          "0x87a75af70c563b78598434d65dfdeca7eabd98f5f75a68281216ea40ff15648a",
		fromPublicKey: "0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5",
		toPublicKey:   "0xeBec795c9c8bBD61FFc14A6662944748F299cAcf",
	}} {
		policyID, err := uuid.NewUUID()
		require.Nil(t, err)

		txBefore, err := db.CreateTx(ctx, storage.CreateTxDto{
			PluginID:      "vultisig-payroll-0000",
			ChainID:       testcase.chain,
			PolicyID:      policyID,
			FromPublicKey: testcase.fromPublicKey,
			ToPublicKey:   testcase.toPublicKey,
			ProposedTxHex: "0x1",
		})
		require.Nil(t, err, testcase.chain.String())
		require.Equal(t, storage.TxProposed, txBefore.Status, testcase.chain.String())
		var nilOnChainStatus *rpc.TxOnChainStatus
		require.Equal(t, nilOnChainStatus, txBefore.StatusOnChain, testcase.chain.String())

		err = db.SetSignedAndBroadcasted(ctx, txBefore.ID, testcase.hash)
		require.Nil(t, err, testcase.chain.String())

		err = worker.updatePendingTxs()
		require.Nil(t, err, testcase.chain.String())

		txAfter, err := db.GetTxByID(ctx, txBefore.ID)
		require.Nil(t, err, testcase.chain.String())

		require.Equal(t, storage.TxSigned, txAfter.Status, testcase.chain.String())
		require.Equal(t, conv.Ptr(rpc.TxOnChainSuccess), txAfter.StatusOnChain, testcase.chain.String())
	}
}

// TestWorker_failedTransaction tests error message extraction for failed EVM transactions.
// Uses real failed transactions from Sepolia testnet. Requires RPC_ETHEREUM_URL pointing to Sepolia.
func TestWorker_failedTransaction(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		return
	}

	ctx := context.Background()

	worker, stop, db, createErr := createWorker()
	require.Nil(t, createErr)
	defer stop()

	type suite struct {
		name                 string
		chain                common.Chain
		hash                 string
		fromPublicKey        string
		toPublicKey          string
		expectedStatus       rpc.TxOnChainStatus
		expectedErrorContain string
	}

	for _, testcase := range []suite{
		{
			name:                 "out_of_gas",
			chain:                common.Ethereum,
			hash:                 "0x93914a5ed4244d24f6a5570dfd6cbf5dfc8d5f6083e6691072a52e25250a7fd4",
			fromPublicKey:        "0xdF918324C0BBa4BA463f208328451C5710311a65",
			toPublicKey:          "0xd8A62e777714535c9A3006872661263a825F8803",
			expectedStatus:       rpc.TxOnChainFail,
			expectedErrorContain: "out of gas",
		},
		{
			name:                 "transaction_reverted",
			chain:                common.Ethereum,
			hash:                 "0x66b5d3e2a830b0fd10439112d60e0240cea042b6f4bd11eac6721116cf0b4020",
			fromPublicKey:        "0xD7D771d3024A3d6C7CaEaF669048D54cD1a0C3c4",
			toPublicKey:          "0x81027470d5626e93C31935d9c7666F5392464943",
			expectedStatus:       rpc.TxOnChainFail,
			expectedErrorContain: "transaction reverted",
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			policyID, err := uuid.NewUUID()
			require.Nil(t, err)

			txBefore, err := db.CreateTx(ctx, storage.CreateTxDto{
				PluginID:      "vultisig-test-0000",
				ChainID:       testcase.chain,
				PolicyID:      policyID,
				FromPublicKey: testcase.fromPublicKey,
				ToPublicKey:   testcase.toPublicKey,
				ProposedTxHex: "0x1",
			})
			require.Nil(t, err)

			err = db.SetSignedAndBroadcasted(ctx, txBefore.ID, testcase.hash)
			require.Nil(t, err)

			err = worker.updatePendingTxs()
			require.Nil(t, err)

			txAfter, err := db.GetTxByID(ctx, txBefore.ID)
			require.Nil(t, err)

			require.Equal(t, conv.Ptr(testcase.expectedStatus), txAfter.StatusOnChain)
			require.NotNil(t, txAfter.ErrorMessage)
			require.Contains(t, *txAfter.ErrorMessage, testcase.expectedErrorContain)

			t.Logf("Hash: %s, Status: %v, ErrorMessage: %s",
				testcase.hash,
				*txAfter.StatusOnChain,
				*txAfter.ErrorMessage)
		})
	}
}
