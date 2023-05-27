package task

import (
	"context"

	"github.com/sunyakun/gearbox/pkg/apis"
)

type TaskName string

type Task interface {
	Do(ctx context.Context, input apis.RawExtension) error
}

type TaskFunc func(ctx context.Context, input apis.RawExtension) error

func (tf TaskFunc) Do(ctx context.Context, input apis.RawExtension) error {
	return tf(ctx, input)
}

type GenericTask[T any] interface {
	Do(ctx context.Context, input T) error
}

type TaskConfig struct {
	AckLate bool
}

type TaskDescriber struct {
	TaskName TaskName
	Func     TaskFunc
	Config   TaskConfig
}

type TaskHub interface {
	Register(TaskDescriber) (TaskName, error)
	MustRegister(TaskDescriber) TaskName
	Emit(name TaskName, input apis.RawExtension) error
	RunForever(context.Context) error
}
