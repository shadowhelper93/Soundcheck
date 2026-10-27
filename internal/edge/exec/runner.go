// Package exec provides process execution, streaming, and lifecycle management.
package exec

import (
	"context"
	"time"
)

// Cmd defines the specification for executing an external command.
type Cmd struct {
	Binary  string
	Args    []string
	Env     []string
	Dir     string
	Timeout time.Duration
}

// Result captures the outcome of a command execution.
type Result struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
	Duration time.Duration
	TimedOut bool
	Killed   bool
}

// EventType identifies the kind of streaming output event.
type EventType string

const (
	// EventStdout indicates data received on standard output.
	EventStdout EventType = "stdout"
	// EventStderr indicates data received on standard error.
	EventStderr EventType = "stderr"
	// EventExit indicates process completion.
	EventExit EventType = "exit"
	// EventOutput indicates data received from an interactive terminal (PTY).
	EventOutput EventType = "output"
	// EventResize indicates a terminal window resize event.
	EventResize EventType = "resize"
)

// Event represents a single chunk or termination signal from an executing command.
type Event struct {
	Type     EventType
	Data     []byte
	ExitCode int
	Error    error
	TimedOut bool
	Killed   bool
}

// Runner defines the contract for executing processes synchronously and streaming.
type Runner interface {
	Run(ctx context.Context, cmd Cmd) (Result, error)
	Stream(ctx context.Context, cmd Cmd) (<-chan Event, error)
}
