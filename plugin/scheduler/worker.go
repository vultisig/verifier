package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/vultisig/verifier/plugin/metrics"
	"github.com/vultisig/verifier/plugin/safety"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/graceful"
)

type Worker struct {
	logger  *logrus.Logger
	metrics metrics.SchedulerMetrics

	client *asynq.Client
	task   string
	queue  string

	repo     Storage
	interval Interval
	policy   PolicyFetcher
	safety   SafetyManager

	pollInterval     time.Duration
	iterationTimeout time.Duration
}

func NewWorker(
	logger *logrus.Logger,
	client *asynq.Client,
	task,
	queue string,
	repo Storage,
	interval Interval,
	policy PolicyFetcher,
	schedulerMetrics metrics.SchedulerMetrics,
	safety SafetyManager,
) *Worker {
	return &Worker{
		logger:           logger.WithField("pkg", "scheduler.Worker").Logger,
		metrics:          schedulerMetrics,
		client:           client,
		task:             task,
		queue:            queue,
		repo:             repo,
		interval:         interval,
		policy:           policy,
		safety:           safety,
		pollInterval:     30 * time.Second,
		iterationTimeout: 30 * time.Second,
	}
}

func (w *Worker) Run() error {
	ctx, stop := context.WithCancel(context.Background())

	go func() {
		graceful.HandleSignals(stop)
		w.logger.Info("got exit signal, will stop after current processing step finished...")
	}()

	err := w.start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start: %w", err)
	}
	return nil
}

func (w *Worker) start(aliveCtx context.Context) error {
	err := w.enqueue()
	if err != nil {
		return fmt.Errorf("failed to enqueue: %w", err)
	}

	for {
		select {
		case <-aliveCtx.Done():
			w.logger.Infof("context done & no processing: stop worker")
			return nil
		case <-time.After(w.pollInterval):
			er := w.enqueue()
			if er != nil {
				w.logger.Errorf("processing error, continue loop: %v", er)
			}
		}
	}
}

func (w *Worker) enqueue() error {
	ctx, cancel := context.WithTimeout(context.Background(), w.iterationTimeout)
	defer cancel()

	w.logger.Info("worker tick")

	tasks, err := w.repo.GetPending(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending tasks: %w", err)
	}

	// Collect metrics
	w.collectMetrics(tasks)

	var eg errgroup.Group
	for _, _task := range tasks {
		task := _task
		eg.Go(func() error {
			policy, err := w.policy.GetPluginPolicy(ctx, task.PolicyID)
			if err != nil {
				return fmt.Errorf("failed to fetch policy: %w", err)
			}

			if w.safety != nil {
				err = w.safety.EnforceKeysign(ctx, string(policy.PluginID))
				if err != nil {
					if safety.IsDisabledError(err) {
						w.logger.WithFields(logrus.Fields{
							"plugin_id": policy.PluginID,
							"id":        policy.ID,
						}).Info("deactivating policy: plugin is paused")
						err = w.repo.Delete(ctx, task.PolicyID)
						if err != nil {
							return fmt.Errorf("failed to delete schedule: %w", err)
						}
						policy.Active = false
						_, err = w.policy.UpdatePluginPolicy(ctx, *policy)
						if err != nil {
							return fmt.Errorf("failed to deactivate policy: %w", err)
						}
						return nil
					}
					w.logger.WithField("plugin_id", policy.PluginID).
						Errorf("failed to check safety: %v", err)
					return fmt.Errorf("safety check failed: %w", err)
				}
			}

			next, err := w.interval.FromNowWhenNext(*policy)
			if err != nil {
				return fmt.Errorf("failed to compute next: %w", err)
			}

			buf, err := json.Marshal(task)
			if err != nil {
				return fmt.Errorf("failed to marshal task: %w", err)
			}

			_, err = w.client.EnqueueContext(
				ctx,
				asynq.NewTask(w.task, buf),
				asynq.MaxRetry(0),
				asynq.Timeout(5*time.Minute),
				asynq.Retention(10*time.Minute),
				asynq.Queue(w.queue),
			)
			if err != nil {
				return fmt.Errorf("failed to enqueue task: %w", err)
			}

			if next.IsZero() {
				err = w.repo.Delete(ctx, task.PolicyID)
				if err != nil {
					return fmt.Errorf("failed to delete schedule: %w", err)
				}
				policy.Active = false
				_, err = w.policy.UpdatePluginPolicy(ctx, *policy)
				if err != nil {
					return fmt.Errorf("failed to deactivate policy: %w", err)
				}
				w.logger.Infof("policy_id=%s: deactivated (no more executions)", task.PolicyID)
				return nil
			}

			err = w.repo.SetNext(ctx, task.PolicyID, next)
			if err != nil {
				return fmt.Errorf("failed to set next: %w", err)
			}
			return nil
		})
	}
	err = eg.Wait()
	if err != nil {
		return fmt.Errorf("failed to process tasks: %w", err)
	}

	return nil
}

func (w *Worker) collectMetrics(tasks []Scheduler) {
	if w.metrics == nil {
		return
	}

	activePolicies := float64(len(tasks))
	w.metrics.SetActivePolicies(activePolicies)

	now := time.Now()
	stuckCount := 0
	for _, task := range tasks {
		if task.NextExecution.Before(now) {
			stuckCount++
		}
	}

	w.metrics.SetStuckPolicies(float64(stuckCount))
}
