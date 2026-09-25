package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	servicesdomain "aether/internal/modules/services/domain"
	"aether/internal/platform/druntime/queue"
)

type ServiceLifecycleStore interface {
	GetLifecycleOperation(context.Context, uuid.UUID) (*servicesdomain.LifecycleOperation, error)
	ClaimLifecycleOperation(context.Context, uuid.UUID) (*servicesdomain.LifecycleOperation, bool, error)
	RenewLifecycleOperation(context.Context, uuid.UUID, int) error
	CompleteLifecycleOperation(context.Context, uuid.UUID, int, string, string) (*servicesdomain.LifecycleOperation, error)
	RecoverLifecycleOperations(context.Context, int) error
}

type ServiceLifecycleNotifier interface {
	NotifyServiceState(context.Context, uuid.UUID, uuid.UUID, string)
}

type ServiceLifecycleMetrics interface {
	StartJob(string) func(bool)
}

type ServiceLifecycleWorker struct {
	Queue         queue.Queue
	QueueStream   string
	ConsumerGroup string
	Store         ServiceLifecycleStore
	Runtime       Runtime
	Execute       func(context.Context, *servicesdomain.LifecycleOperation) error
	Notifier      ServiceLifecycleNotifier
	Metrics       ServiceLifecycleMetrics
	Logger        *slog.Logger
}

func (w *ServiceLifecycleWorker) Run(ctx context.Context) {
	if w.Queue == nil {
		w.log(ctx, "service lifecycle queue consumer", errors.New("service lifecycle queue is not configured"))
		return
	}
	go w.recoverOperations(ctx)
	consumerGroup := w.ConsumerGroup
	if consumerGroup == "" {
		consumerGroup = "service-lifecycle-workers"
	}
	queueStream := w.QueueStream
	if queueStream == "" {
		queueStream = "service-operations"
	}
	for ctx.Err() == nil {
		consumer, err := w.Queue.NewConsumer(ctx, queueStream, consumerGroup, "aether-service-lifecycle")
		if err != nil {
			w.log(ctx, "service lifecycle queue consumer", err)
			if !waitForLifecycleConsumerRetry(ctx) {
				return
			}
			continue
		}
		for ctx.Err() == nil {
			job, nextErr := consumer.Next(ctx)
			if nextErr != nil {
				if ctx.Err() == nil {
					w.log(ctx, "service lifecycle queue next", nextErr)
				}
				break
			}
			w.process(ctx, consumer, job)
		}
		_ = consumer.Close()
		if ctx.Err() == nil && !waitForLifecycleConsumerRetry(ctx) {
			return
		}
	}
}

func waitForLifecycleConsumerRetry(ctx context.Context) bool {
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (w *ServiceLifecycleWorker) process(ctx context.Context, consumer queue.Consumer, job *queue.Job) {
	var payload struct {
		OperationID string `json:"operation_id"`
	}
	if job == nil {
		return
	}
	if json.Unmarshal(job.Payload, &payload) != nil {
		_ = consumer.Ack(ctx, job)
		return
	}
	operationID, err := uuid.Parse(payload.OperationID)
	if err != nil {
		_ = consumer.Ack(ctx, job)
		return
	}
	operation, claimed, err := w.Store.ClaimLifecycleOperation(ctx, operationID)
	if err != nil {
		w.log(ctx, "claim service lifecycle operation", err)
		_ = consumer.Nack(ctx, job)
		return
	}
	if !claimed {
		operation, getErr := w.Store.GetLifecycleOperation(ctx, operationID)
		if errors.Is(getErr, servicesdomain.ErrNotFound) || getErr == nil && (operation.Status == "succeeded" || operation.Status == "failed") {
			_ = consumer.Ack(ctx, job)
			return
		}
		_ = consumer.Nack(ctx, job)
		return
	}
	metricFailed := true
	finishMetrics := func(bool) {}
	if w.Metrics != nil {
		finishMetrics = w.Metrics.StartJob("service.lifecycle." + string(operation.Action))
	}
	defer func() { finishMetrics(metricFailed) }()
	if w.Notifier != nil {
		w.Notifier.NotifyServiceState(ctx, operation.OrgID, operation.ServiceID, activeLifecycleStatus(operation.Action))
	}
	stopProgress := queue.StartProgress(ctx, consumer, job)
	leaseCtx, stopLease := context.WithCancel(ctx)
	go w.renewLease(leaseCtx, operationID, operation.Attempts)
	if w.Logger != nil {
		w.Logger.InfoContext(ctx, "service lifecycle operation started", "service_id", operation.ServiceID, "operation_id", operation.ID, "command_id", job.ID, "operation", operation.Action, "accepted_state", activeLifecycleStatus(operation.Action))
	}
	finalStatus, executionErr := w.executeAndVerify(ctx, operation)
	stopLease()
	stopProgress()
	if ctx.Err() != nil {
		_ = consumer.Nack(context.Background(), job)
		return
	}
	operationError := ""
	if executionErr != nil {
		operationError = "runtime operation failed; observed state: " + finalStatus
		if w.Logger != nil {
			w.Logger.ErrorContext(ctx, "service lifecycle runtime operation failed", "service_id", operation.ServiceID, "operation_id", operation.ID, "command_id", job.ID, "operation", operation.Action, "observed_state", finalStatus, "failure_type", fmt.Sprintf("%T", executionErr))
		}
	}
	completed, err := w.Store.CompleteLifecycleOperation(ctx, operationID, operation.Attempts, finalStatus, operationError)
	if err != nil {
		w.log(ctx, "complete service lifecycle operation", err)
		_ = consumer.Nack(ctx, job)
		return
	}
	if completed.Status == "succeeded" || completed.Status == "failed" {
		metricFailed = completed.Status != "succeeded"
		if completed.Status == "succeeded" {
			if completed.Action == servicesdomain.LifecycleStart {
				finalStatus = string(servicesdomain.StatusRunning)
			} else {
				finalStatus = string(servicesdomain.StatusStopped)
			}
		}
		if w.Notifier != nil {
			w.Notifier.NotifyServiceState(ctx, completed.OrgID, completed.ServiceID, finalStatus)
		}
		if w.Logger != nil {
			w.Logger.InfoContext(ctx, "service lifecycle operation completed", "service_id", completed.ServiceID, "operation_id", completed.ID, "command_id", job.ID, "operation", completed.Action, "next_state", finalStatus, "duration", time.Since(operationStartTime(operation)).String(), "result", completed.Status)
		}
	}
	_ = consumer.Ack(ctx, job)
}

func (w *ServiceLifecycleWorker) renewLease(ctx context.Context, operationID uuid.UUID, attempt int) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.Store.RenewLifecycleOperation(ctx, operationID, attempt); err != nil {
				w.log(ctx, "renew service lifecycle operation lease", err)
				return
			}
		}
	}
}

func (w *ServiceLifecycleWorker) recoverOperations(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.Store.RecoverLifecycleOperations(ctx, 32); err != nil {
				w.log(ctx, "recover stale service lifecycle operations", err)
			}
		}
	}
}

func (w *ServiceLifecycleWorker) executeAndVerify(ctx context.Context, operation *servicesdomain.LifecycleOperation) (string, error) {
	if w.Execute == nil || w.Runtime == nil {
		return string(servicesdomain.StatusUnknown), errors.New("service lifecycle executor is not configured")
	}
	executionErr := w.Execute(ctx, operation)
	containers, err := w.Runtime.ListContainers(ctx)
	if err != nil {
		return string(servicesdomain.StatusUnknown), errors.Join(executionErr, fmt.Errorf("verify service runtime state: %w", err))
	}
	serviceContainers := make([]ContainerInfo, 0)
	for _, container := range containers {
		if container.Labels["aether.service-id"] == operation.ServiceID.String() {
			serviceContainers = append(serviceContainers, container)
		}
	}
	if len(serviceContainers) == 0 && operation.Action == servicesdomain.LifecycleStop {
		return string(servicesdomain.StatusStopped), nil
	}
	if len(serviceContainers) == 0 {
		return string(servicesdomain.StatusFailed), errors.Join(executionErr, errors.New("service runtime did not report any containers after lifecycle operation"))
	}
	runningContainers := 0
	for _, container := range serviceContainers {
		if isRunningContainerState(container.State) {
			runningContainers++
		}
	}
	if operation.Action == servicesdomain.LifecycleStart && runningContainers == len(serviceContainers) {
		return string(servicesdomain.StatusRunning), nil
	}
	if operation.Action == servicesdomain.LifecycleStop && runningContainers == 0 {
		return string(servicesdomain.StatusStopped), nil
	}
	observedStatus := string(servicesdomain.StatusDegraded)
	if runningContainers == len(serviceContainers) {
		observedStatus = string(servicesdomain.StatusRunning)
	}
	if executionErr == nil {
		executionErr = fmt.Errorf("runtime did not reach requested state")
	}
	return observedStatus, executionErr
}

func isRunningContainerState(state string) bool {
	return strings.EqualFold(strings.TrimSpace(state), "running")
}

func activeLifecycleStatus(action servicesdomain.LifecycleAction) string {
	if action == servicesdomain.LifecycleStart {
		return string(servicesdomain.StatusStarting)
	}
	return string(servicesdomain.StatusStopping)
}

func operationStartTime(operation *servicesdomain.LifecycleOperation) time.Time {
	if operation.LeaseUntil != nil {
		return operation.LeaseUntil.Add(-45 * time.Second)
	}
	return time.Now()
}

func (w *ServiceLifecycleWorker) log(ctx context.Context, message string, err error) {
	if w.Logger != nil {
		w.Logger.ErrorContext(ctx, message, "failure_type", fmt.Sprintf("%T", err))
	}
}
