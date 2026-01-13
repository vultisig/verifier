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

	eg := &errgroup.Group{}
	for _, _task := range tasks {
		task := _task
		eg.Go(func() error {
			policy, er := w.policy.GetPluginPolicy(ctx, task.PolicyID)
			if er != nil {
				return fmt.Errorf("failed to fetch policy: %w", er)
			}

			if w.safety != nil {
				er = w.safety.EnforceKeysign(ctx, string(policy.PluginID))
				if er != nil {
					if safety.IsDisabledError(er) {
						w.logger.WithField("plugin_id", policy.PluginID).
							Info("skipping enqueue: plugin is paused")
						return nil
					}
					w.logger.WithField("plugin_id", policy.PluginID).
						Errorf("failed to check safety: %v", er)
					return fmt.Errorf("safety check failed: %w", er)
				}
			}

			next, er := w.interval.FromNowWhenNext(*policy)
			if er != nil {
				return fmt.Errorf("failed to compute next: %w", er)
			}

			buf, er := json.Marshal(task)
			if er != nil {
				return fmt.Errorf("failed to marshal task: %w", er)
			}

			_, er = w.client.EnqueueContext(
				ctx,
				asynq.NewTask(w.task, buf),
				asynq.MaxRetry(0),
				asynq.Timeout(5*time.Minute),
				asynq.Retention(10*time.Minute),
				asynq.Queue(w.queue),
			)
			if er != nil {
				return fmt.Errorf("failed to enqueue task: %w", er)
			}

			if next.IsZero() {
				// Delete from scheduler
				e := w.repo.Delete(ctx, task.PolicyID)
				if e != nil {
					return fmt.Errorf("failed to delete schedule: %w", e)
				}

				// Set policy active = false
				policy.Active = false
				_, e = w.policy.UpdatePluginPolicy(ctx, *policy)
				if e != nil {
					return fmt.Errorf("failed to deactivate policy: %w", e)
				}
				w.logger.Infof("policy_id=%s: deactivated (no more executions)", task.PolicyID)
				return nil
			}

			er = w.repo.SetNext(ctx, task.PolicyID, next)
			if er != nil {
				return fmt.Errorf("failed to set next: %w", er)
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
