package tasks

import (
	"fmt"
	"os"

	"github.com/hibiken/asynq"
)

const defaultQueueName = "default_queue"

var QUEUE_NAME = getQueueName()

func getQueueName() string {
	if name := os.Getenv("TASK_QUEUE_NAME"); name != "" {
		return name
	}
	return defaultQueueName
}

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
