package tx_indexer

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/vultisig/verifier/common"
	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/conv"
	"github.com/vultisig/verifier/internal/rpc"
	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/internal/storage/postgres"
	"github.com/vultisig/verifier/internal/types"
	"os"
	"testing"
)

func createWorker() (*Worker, context.CancelFunc, storage.TxIndexerRepository, error) {
	ctx, stop := context.WithCancel(context.Background())

	logger := logrus.New()

	cfg, err := config.ReadTxIndexerConfig()
	if err != nil {
		return nil, stop, nil, fmt.Errorf("config.ReadTxIndexerConfig: %w", err)
	}

	rpcBtc, err := rpc.NewBitcoinClient(cfg.Rpc.Bitcoin.URL)
	if err != nil {
		return nil, stop, nil, fmt.Errorf("rpc.NewBitcoinClient: %w", err)
	}

	rpcEth, err := rpc.NewEvmClient(ctx, cfg.Rpc.Ethereum.URL)
	if err != nil {
		return nil, stop, nil, fmt.Errorf("rpc.NewEvmClient: %w", err)
	}

	db, err := postgres.NewPostgresBackend(cfg.Database.DSN, nil)
	if err != nil {
		return nil, stop, nil, fmt.Errorf("postgres.NewPostgresBackend: %w", err)
	}

	worker := NewWorker(
		logger,
		cfg.Interval,
		cfg.IterationTimeout,
		cfg.MarkLostAfter,
		cfg.Concurrency,
		db,
		map[common.Chain]rpc.TxIndexer{
			common.Bitcoin:  rpcBtc,
			common.Ethereum: rpcEth,
		},
	)

	return worker, stop, db, nil
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
	}
	for _, testcase := range []suite{{
		chain:         common.Bitcoin,
		hash:          "75149b57b808eb2083bada72018c48118238427a229e75896e30ccd09646df8e",
		fromPublicKey: "1Hk4sk6kNscQ32ZZ9eGMtM2URNbYYRXLtX",
	}, {
		chain:         common.Ethereum,
		hash:          "0x87a75af70c563b78598434d65dfdeca7eabd98f5f75a68281216ea40ff15648a",
		fromPublicKey: "0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5",
	}} {
		pluginID, err := uuid.NewUUID()
		require.Nil(t, err)
		policyID, err := uuid.NewUUID()
		require.Nil(t, err)

		txBefore, err := db.CreateTx(ctx, types.CreateTxDto{
			PluginID:         pluginID,
			ChainID:          testcase.chain,
			PolicyID:         policyID,
			FromPublicKey:    testcase.fromPublicKey,
			ProposedTxObject: []byte(`{}`),
		})
		require.Nil(t, err, testcase.chain.String())
		require.Equal(t, types.TxProposed, txBefore.Status, testcase.chain.String())
		var nilOnChainStatus *types.TxOnChainStatus
		require.Equal(t, nilOnChainStatus, txBefore.StatusOnChain, testcase.chain.String())

		err = db.SetSignedAndBroadcasted(ctx, txBefore.ID, testcase.hash)
		require.Nil(t, err, testcase.chain.String())

		err = worker.updatePendingTxs()
		require.Nil(t, err, testcase.chain.String())

		txAfter, err := db.GetTxByID(ctx, txBefore.ID)
		require.Nil(t, err, testcase.chain.String())

		require.Equal(t, types.TxSigned, txAfter.Status, testcase.chain.String())
		require.Equal(t, conv.Ptr(types.TxOnChainSuccess), txAfter.StatusOnChain, testcase.chain.String())
	}
}
