package main

import (
	"fmt"
	"slices"
	"testing"
	"time"
)

type Check struct {
	before string
	then   string
}

func (c Check) CheckSlice(sl []string) bool {
	beforeIndex := slices.Index(sl, c.before)
	thenIndex := slices.Index(sl, c.then)
	return beforeIndex != -1 && thenIndex != -1 && beforeIndex < thenIndex
}

func (c Check) String() string {
	return fmt.Sprintf("%s before %s", c.before, c.then)
}

type TestTask struct {
	deps []string
	name string
	ch   chan string
}

// DependesOn implements Task.
func (tt TestTask) DependsOn() []string {
	return tt.deps
}

// GetKey implements Task.
func (tt TestTask) GetKey() string {
	return tt.name
}

// Run implements Task.
func (tt TestTask) Run() error {
	tt.ch <- tt.name
	time.Sleep(1 * time.Second)
	return nil
}

var _ Task[string] = TestTask{}

func TestTaskRunner(t *testing.T) {
	ch := make(chan string, 5)
	defer close(ch)
	tasks := []TestTask{
		{
			name: "A",
			deps: []string{},
			ch:   ch,
		},
		{
			name: "B",
			deps: []string{},
			ch:   ch,
		},
		{
			name: "C",
			deps: []string{"A", "B"},
			ch:   ch,
		},
		{
			name: "D",
			deps: []string{"C"},
			ch:   ch,
		},
	}

	taskRunner := NewTaskRunner(tasks)

	err := taskRunner.Run()
	if err != nil {
		t.Error(err)
	}

	done := make([]string, len(tasks))
	for i := 0; i < len(tasks); i++ {
		done = append(done, <-ch)
	}

	t.Log(done)

	checks := []Check{
		{"A", "C"},
		{"B", "C"},
		{"C", "D"},
	}
	for _, check := range checks {
		t.Run(check.String(), func(t *testing.T) {
			if !check.CheckSlice(done) {
				t.Fail()
			}
		})
	}
}
