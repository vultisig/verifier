package tx_indexer

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/verifier/internal/types"
	"time"
)

type Worker struct {
	logger           *logrus.Logger
	interval         time.Duration
	iterationTimeout time.Duration
	lostTimeout      time.Duration
	concurrency      int
	clients          map[types.Chain]Client
}

func NewWorker(
	logger *logrus.Logger,
	interval time.Duration,
	iterationTimeout time.Duration,
	lostTimeout time.Duration,
	concurrency int,
	clients map[types.Chain]Client,
) *Worker {
	return &Worker{
		logger:           logger,
		interval:         interval,
		iterationTimeout: iterationTimeout,
		lostTimeout:      lostTimeout,
		concurrency:      concurrency,
		clients:          clients,
	}
}

func (w *Worker) Start(aliveCtx context.Context) error {
	for {
		select {
		case <-aliveCtx.Done():
			w.logger.Infof("context done & no processing: stop worker")
			return nil
		case <-time.After(w.interval):
			err := w.do()
			if err != nil {
				w.logger.Errorf("processing error, continue loop: %v", err)
			}
		}
	}
}

func (w *Worker) do() error {
	ctx, cancel := context.WithTimeout(context.Background(), w.iterationTimeout)
	defer cancel()

	return nil
}
