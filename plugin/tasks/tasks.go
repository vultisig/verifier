package tasks

import (
	"fmt"

	"github.com/hibiken/asynq"
)

const QUEUE_NAME = "default_queue"
const (
	TypeRecurringFeeRecord = "fee:recurringRecord"
	TypePluginTransaction  = "plugin:transaction"
	TypeKeyGenerationDKLS  = "key:generationDKLS"
	TypeKeySignDKLS        = "key:signDKLS"
	TypeReshareDKLS        = "key:reshareDKLS"
)

func GetTaskResult(inspector *asynq.Inspector, taskID string) ([]byte, error) {
	task, err := inspector.GetTaskInfo(QUEUE_NAME, taskID)
	if err != nil {
		return nil, fmt.Errorf("fail to find task, err: %w", err)
	}

	if task == nil {
		return nil, fmt.Errorf("task not found")
	}

	if task.State == asynq.TaskStatePending {
		return nil, fmt.Errorf("task is still in progress")
	}

	if task.State == asynq.TaskStateCompleted {
		return task.Result, nil
	}

	return nil, fmt.Errorf("task state is invalid")
}
