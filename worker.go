package skyflowclient

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/panjf2000/ants/v2"
	pbv1 "github.com/skyflow-workflow/skyflow_backend/api/v1"
)

var DefaultGetActivityTaskTimeout = 10 * time.Second
var DefaultPollInterval = 200 * time.Millisecond
var DefaultPollErrorInterval = 1 * time.Second

// ActivityFunction activity callback function
type ActivityFunction func(ctx *Context) error

// WorkerActivity Activity in worker
type WorkerActivity struct {
	Name        string           //  活动名称
	Function    ActivityFunction // 指定的函数
	Description string           // 活动说明
	URI         string           // 活动URI
}

// ActivityTaskRuntime    一个具体的Activity 运行时实例
type ActivityTaskRuntime struct {
	activity *WorkerActivity
	data     *pbv1.GetActivityTaskResponse
}

// ActivityWorker Worker for activity model
type ActivityWorker struct {
	client                 *Client
	Namespaces             []*WorkerNamespace
	stop                   bool
	workerPool             *ants.PoolWithFunc
	poolsize               int
	status                 WorkerStatusType //worker的状态
	GetActivityTaskTimeout time.Duration    // 获取活动任务超时时间
	PollInterval           time.Duration    // 正常轮询间隔
	PollErrorInterval      time.Duration    //轮询错误间隔
}

// WorkerStatusType  worker 状态类型
type WorkerStatusType string

// WorkerStatus worker的状态分布
var WorkerStatus = struct {
	Init       WorkerStatusType
	Stop       WorkerStatusType
	StopFinish WorkerStatusType
	Running    WorkerStatusType
	Close      WorkerStatusType
}{
	// 初始状态
	Init:       "Init",
	Stop:       "Stop",
	StopFinish: "StopFinish",
	Running:    "Running",
	Close:      "Close",
}

// ActivityRunError  运行时错误类型
var ActivityRunError = "ActivityRunError"

// NewActivityWorker new activity worker
func NewActivityWorker(client *Client, poolsize int, namespaces ...*WorkerNamespace) (*ActivityWorker, error) {
	var err error
	aw := ActivityWorker{
		client:                 client,
		Namespaces:             namespaces,
		stop:                   false,
		status:                 WorkerStatus.Init,
		poolsize:               poolsize,
		GetActivityTaskTimeout: DefaultGetActivityTaskTimeout,
		PollInterval:           DefaultPollInterval,
		PollErrorInterval:      DefaultPollErrorInterval,
	}
	workerPool, err := ants.NewPoolWithFunc(poolsize, aw.runActivity)
	if err != nil {
		return nil, err
	}
	aw.workerPool = workerPool
	return &aw, nil
}

func (w *ActivityWorker) monitorActivity(activity *WorkerActivity) {

	for {
		if w.stop {
			break
		}
		GetActivitiesTaskResp, err := w.client.GetActivityTask(
			context.Background(),
			&pbv1.GetActivityTaskRequest{
				ActivityUri: activity.URI,
			})
		if err != nil {
			// transport error to kratos error
			kerr := errors.FromError(err)

			// if error code is ActivityNotFound, then sleep
			if kerr.Code == int32(pbv1.ErrorCode_ACTIVITY_NOT_FOUND) {
				time.Sleep(w.PollInterval)
				continue
			}
			// other error, then sleep
			time.Sleep(w.PollErrorInterval)
			continue
		}

		data := ActivityTaskRuntime{
			activity: activity,
			data:     GetActivitiesTaskResp,
		}
		err = w.workerPool.Invoke(data)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

	}

}

func (w *ActivityWorker) runActivity(i interface{}) {

	var err error
	actrun, ok := i.(ActivityTaskRuntime)
	if !ok {
		return
	}
	var token = actrun.data.TaskToken
	defer func() {

		var r = recover()
		if r != nil {
			message := fmt.Sprint(r)
			err = w.client.SendTaskFailure(context.Background(), &pbv1.SendTaskFailureRequest{
				TaskToken: token,
				Error:     ActivityRunError,
				Cause:     message,
			})
			if err != nil {
				slog.ErrorContext(context.Background(), "send task failure error", "error", err)
				return
			}
		}
	}()

	ctx := NewContext(context.Background(), w.client, &actrun)

	err = actrun.activity.Function(ctx)
	if err != nil {
		err = ctx.SendTaskFailure(ActivityRunError, err.Error())
		slog.ErrorContext(ctx.Context, "send task failure error", "error", err)
	}
}

// Run  Start Monitor skyflow and execute function when need
func (w *ActivityWorker) Run() error {
	var err error
	err = w.Register()
	if err != nil {
		return err
	}

	for _, ns := range w.Namespaces {
		for _, act := range ns.Activities {
			go w.monitorActivity(act)
		}
	}
	w.status = WorkerStatus.Running
	return nil
}

// Register register statemachine / activities to skyflow server
func (w *ActivityWorker) Register() error {

	var err error
	for _, ns := range w.Namespaces {
		err = ns.ScanStateMachinePath()
		if err != nil {
			return err
		}
		_, err = w.client.CreateOrUpdateNamespace(
			context.Background(),
			&pbv1.CreateNamespaceRequest{
				Name:        ns.Name,
				Description: ns.Description,
			})
		if err != nil {
			return err
		}
		for _, act := range ns.Activities {
			activity, err := w.client.CreateOrUpdateActivity(
				context.Background(),
				&pbv1.CreateActivityRequest{
					Namespace:   ns.Name,
					Name:        act.Name,
					Description: act.Description,
				})
			if err != nil {
				return err
			}
			act.URI = activity.ActivityUri
		}
		for name, content := range ns.StateMachines {
			_, err = w.client.CreateOrUpdateStateMachine(
				context.Background(),
				&pbv1.CreateStateMachineRequest{
					Namespace:   ns.Name,
					Name:        name,
					Description: "statemachine from file",
					Definition:  content,
				})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Stop Stop Monitor skyflow
func (w *ActivityWorker) Stop() error {
	w.stop = true
	w.status = WorkerStatus.Stop
	for {
		if w.workerPool.Running() == 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}
	w.status = WorkerStatus.StopFinish
	return nil
}

// Status 返回worker的状态
func (w *ActivityWorker) Status() WorkerStatusType {
	return w.status
}

// ResourceWorker resource worker use api
type ResourceWorker struct {
}

// WorkerNamespace  repository in worker
type WorkerNamespace struct {
	Name              string
	Description       string
	Activities        []*WorkerActivity
	StateMachinePaths []string
	StateMachines     map[string]string
}

// NewWorkerNamespace New worker namespace
func NewWorkerNamespace(name string, description string, paths []string,
	activities ...*WorkerActivity) *WorkerNamespace {

	wr := WorkerNamespace{
		Name:              name,
		Activities:        activities,
		StateMachinePaths: paths,
		StateMachines:     map[string]string{},
	}
	return &wr
}

// ScanStateMachinePath  add statemachine search path
func (wr *WorkerNamespace) ScanStateMachinePath() error {

	for _, p := range wr.StateMachinePaths {
		fileInfo, err := os.Stat(p)
		if err != nil {
			return err
		}
		if fileInfo.IsDir() {
			// 扫描目录
			fileinfos, err := os.ReadDir(p)
			if err != nil {
				return err
			}
			// 遍历文件
			for _, fi := range fileinfos {
				fmt.Println(fi.Name())
				if !fi.IsDir() {
					fileName := fi.Name()
					fileNamePart := strings.SplitN(strings.TrimSpace(fileName), ".", 2)
					if len(fileNamePart) < 2 {
						continue
					}
					fName := fileNamePart[0]
					fExt := fileNamePart[1]
					if fExt == "json" {
						fullfilename := path.Join(p, fileName)
						content, err := os.ReadFile(fullfilename)
						if err != nil {
							return err
						}
						wr.StateMachines[fName] = string(content)
					}
				}
			}
		} else {
			// 处理文件
			fileName := fileInfo.Name()
			fNamePart := strings.SplitN(strings.TrimSpace(fileName), ".", 2)
			if len(fNamePart) < 2 || fNamePart[1] != "json" {
				err = fmt.Errorf("StateMachine Path [ %s ] file name  should like  xx.json ", p)
				return err
			}
			fName := fNamePart[0]
			fExt := fNamePart[1]
			if fExt == "json" {
				content, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				wr.StateMachines[fName] = string(content)
			}

		}
	}
	return nil
}

// NewWorkerActivity New Worker Activity
func NewWorkerActivity(name string, f ActivityFunction, comment string) *WorkerActivity {
	wa := WorkerActivity{
		Name:        name,
		Function:    f,
		Description: comment,
	}
	return &wa
}
