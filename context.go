package skyflowclient

import (
	"context"
	"encoding/json"

	v1 "github.com/skyflow-workflow/skyflow_backend/api/v1"
)

// Context Skyflow Worker Context
// Context 上下文信息，用于传递给Activity 的参数
type Context struct {
	Context context.Context
	Client  *Client
	Runtime *ActivityTaskRuntime
}

// NewContext create a new context
func NewContext(ctx context.Context, client *Client, task *ActivityTaskRuntime) *Context {
	return &Context{
		Context: ctx,
		Client:  client,
		Runtime: task,
	}
}

func (ctx *Context) SendTaskFailure(errorname string, cause string) error {
	req := &v1.SendTaskFailureRequest{
		TaskToken: ctx.Runtime.Data.TaskToken,
		Error:     errorname,
		Cause:     cause,
	}
	return ctx.Client.SendTaskFailure(ctx.Context, req)
}

func (ctx *Context) SendTaskSuccess(output any) error {
	var result []byte
	result, err := json.Marshal(output)
	if err != nil {
		return err
	}
	req := &v1.SendTaskSuccessRequest{
		TaskToken: ctx.Runtime.Data.TaskToken,
		Output:    string(result),
	}
	return ctx.Client.SendTaskSuccess(ctx.Context, req)
}

func (ctx *Context) SendTaskHeartbeat(message string) error {
	req := &v1.SendTaskHeartbeatRequest{
		TaskToken: ctx.Runtime.Data.TaskToken,
		Message:   message,
	}
	return ctx.Client.SendTaskHeartbeat(ctx.Context, req)
}

func (ctx *Context) UnmarshalInput(v interface{}) error {
	var err error
	data := ctx.Runtime.Data.Input
	err = json.Unmarshal([]byte(data), v)
	if err != nil {
		return err
	}
	return nil
}
