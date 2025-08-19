package tx_indexer

import (
	"context"
	"fmt"
	"os"
	"testing"

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
