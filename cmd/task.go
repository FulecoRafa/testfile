package main

import (
	"context"
	"sync"
)

type Task[K comparable] interface {
	GetKey() K
	DependsOn() []K
	Run() error
}

type taskRunnable[T Task[K], K comparable] struct {
	waitOnChans []chan struct{}
	doneChans   []chan struct{}
	task        *T
}

type TaskRunner[T Task[K], K comparable] struct {
	taskMap map[K]*taskRunnable[T, K]
}

func NewTaskRunner[T Task[K], K comparable](tasks []T) *TaskRunner[T, K] {
	taskMap := make(map[K]*taskRunnable[T, K], len(tasks))
	for _, task := range tasks {
		localTask := task
		taskMap[task.GetKey()] = &taskRunnable[T, K]{
			task: &localTask,
		}
	}
	for _, task := range tasks {
		for _, dep := range task.DependsOn() {
			depChan := make(chan struct{}, 1)
			taskMap[task.GetKey()].waitOnChans = append(taskMap[task.GetKey()].waitOnChans, depChan)
			taskMap[dep].doneChans = append(taskMap[dep].doneChans, depChan)
		}
	}

	// Check for cycles

	return &TaskRunner[T, K]{
		taskMap,
	}
}

func (tr *TaskRunner[T, K]) Run() error {
	ctx, cancel := context.WithCancelCause(context.Background())
	doneChan := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(len(tr.taskMap))
	go func() {
		for _, task := range tr.taskMap {
			go tr.RunTask(task, &wg, cancel, ctx.Done())
		}
		wg.Wait()
		doneChan <- struct{}{}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-doneChan:
		return nil
	}
}

func (tr *TaskRunner[T, K]) RunTask(t *taskRunnable[T, K], wg *sync.WaitGroup, cancel context.CancelCauseFunc, isCanceled <-chan struct{}) {
	for _, ch := range t.waitOnChans {
		select {
		case <-ch:
			continue
		case <-isCanceled:
			return
		}
	}
	err := (*t.task).Run()
	if err != nil && cancel != nil {
		cancel(err)
	}
	for _, ch := range t.doneChans {
		ch <- struct{}{}
	}
	wg.Done()
}
