package task

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/sunyakun/gearbox/pkg/apis"
	"github.com/sunyakun/gearbox/pkg/util"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-logr/logr"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

type TaskEntry[T any] struct {
	taskhub  TaskHub
	TaskName TaskName
}

func MustNewTaskEntry[T any](taskhub TaskHub, task GenericTask[T], config TaskConfig) *TaskEntry[T] {
	te := &TaskEntry[T]{taskhub: taskhub}
	te.TaskName = taskhub.MustRegister(CreateTaskDescriber(task, config))
	return te
}

func (t *TaskEntry[T]) Emit(input T) error {
	return t.taskhub.Emit(t.TaskName, apis.RawExtension{Object: input})
}

func CreateTaskDescriber[T any](task GenericTask[T], config TaskConfig) TaskDescriber {
	var (
		rt reflect.Type
	)

	for rt = reflect.TypeOf(task); rt.Kind() == reflect.Pointer; {
		rt = rt.Elem()
	}

	pkgpath := rt.PkgPath()
	if pkgpath == "" {
		pkgpath = "anonymPkg"
	}

	name := rt.Name()
	if name == "" {
		name = "anonymFunc"
	}

	taskName := fmt.Sprintf("%s:%s", pkgpath, name)

	return TaskDescriber{
		TaskName: TaskName(taskName),
		Func: func(ctx context.Context, input apis.RawExtension) error {
			var obj T
			if err := json.Unmarshal(input.Raw, &obj); err != nil {
				return fmt.Errorf("failed to unmarshal RawExtension to %T in call to %s", obj, taskName)
			}
			return task.Do(ctx, obj)
		},
		Config: config,
	}
}

type EmitTaskRequest struct {
	TaskName TaskName          `json:"task_name,omitempty"`
	Input    apis.RawExtension `json:"input,omitempty"`
}

func (p *EmitTaskRequest) Scan(val []byte) error {
	if err := json.Unmarshal(val, &p); err != nil {
		return err
	}
	return nil
}

func (p EmitTaskRequest) Value() ([]byte, error) {
	val, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return val, nil
}

type taskhub struct {
	hubname      string
	logger       logr.Logger
	pub          *sql.Publisher
	sub          *sql.Subscriber
	mu           sync.Mutex
	registry     map[TaskName]TaskDescriber
	crashCatcher func()
	runningMu    sync.RWMutex
	running      bool
}

func NewTaskHub(hubname string, db *gorm.DB, logger logr.Logger) (TaskHub, error) {
	stdsql, err := db.DB()
	if err != nil {
		return nil, err
	}

	schemaAdapter := MySQLSchema{OffsetFieldName: "offset_msg", TaskHubName: hubname}
	offsetSchemaAdapter := MySQLOffsetScheme{TaskHubName: hubname}
	pubConfig := sql.PublisherConfig{
		SchemaAdapter:        schemaAdapter,
		AutoInitializeSchema: false,
	}
	subConfig := sql.SubscriberConfig{
		SchemaAdapter:  schemaAdapter,
		OffsetsAdapter: offsetSchemaAdapter,
	}

	pub, err := sql.NewPublisher(stdsql, pubConfig, watermill.NewStdLogger(false, false))
	if err != nil {
		return nil, err
	}

	sub, err := sql.NewSubscriber(stdsql, subConfig, watermill.NewStdLogger(false, false))
	if err != nil {
		return nil, err
	}

	return &taskhub{
		hubname:  hubname,
		logger:   logger,
		pub:      pub,
		sub:      sub,
		registry: make(map[TaskName]TaskDescriber),
		crashCatcher: util.NewCrashCatcher([]func(err any){
			func(err any) {
				switch e := err.(type) {
				case error:
					logger.Error(e, "panic occurred!")
				default:
					logger.Error(nil, "panic occurred!", "with", e)
				}
			},
		}),
	}, nil
}

func (t *taskhub) Register(desc TaskDescriber) (TaskName, error) {
	err := func() error {
		t.runningMu.RLock()
		defer t.runningMu.RUnlock()
		if t.running {
			return fmt.Errorf("the taskhub already running, register task is forbidden")
		}
		return nil
	}()
	if err != nil {
		return "", err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	_, ok := t.registry[desc.TaskName]
	if ok {
		return "", fmt.Errorf("task %q duplicate", desc.TaskName)
	}
	t.registry[desc.TaskName] = desc
	return TaskName(desc.TaskName), nil
}

func (t *taskhub) MustRegister(desc TaskDescriber) TaskName {
	taskName, err := t.Register(desc)
	if err != nil {
		panic(err)
	}
	return taskName
}

func (t *taskhub) Emit(name TaskName, input apis.RawExtension) error {
	uid := lo.RandomString(8, lo.AlphanumericCharset)
	req := EmitTaskRequest{
		TaskName: name,
		Input:    input,
	}
	payload, err := req.Value()
	if err != nil {
		return err
	}
	return t.pub.Publish(string(name), message.NewMessage(uid, payload))
}

func (t *taskhub) RunForever(ctx context.Context) error {
	var wg sync.WaitGroup

	func() {
		t.runningMu.Lock()
		defer t.runningMu.Unlock()
		t.running = true
	}()

	for taskname, taskDesc := range t.registry {
		desc := taskDesc
		msgCh, err := t.sub.Subscribe(ctx, string(taskname))
		if err != nil {
			return err
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case msg := <-msgCh:
					req := EmitTaskRequest{}
					if err := req.Scan(msg.Payload); err != nil {
						t.logger.Error(err, "scan message payload failed", "payload", msg.Payload)
						continue
					}

					if !desc.Config.AckLate {
						msg.Ack()
					}

					func() {
						defer t.crashCatcher()
						if desc.Config.AckLate {
							defer msg.Ack()
						}
						if err := desc.Func(ctx, req.Input); err != nil {
							t.logger.Error(err, "execute task func failed", "args", req.Input)
						}
					}()
				case <-ctx.Done():
					t.logger.Info("stop process task, context canceled", "taskName", desc.TaskName)
					return
				}
			}
		}()
	}

	wg.Wait()

	func() {
		t.runningMu.Lock()
		defer t.runningMu.Unlock()
		t.running = false
	}()

	return nil
}
