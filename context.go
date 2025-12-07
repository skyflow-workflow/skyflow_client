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
	var err error
	req := &v1.SendTaskFailureRequest{
		TaskToken: ctx.Runtime.data.TaskToken,
		Error:     errorname,
		Cause:     cause,
	}
	_, err = ctx.Client.GetClient().SendTaskFailure(ctx.Context, req)
	if err != nil {
		return err
	}
	return nil
}

func (ctx *Context) SendTaskSuccess(output any) error {
	var err error
	var result []byte
	result, err = json.Marshal(output)
	if err != nil {
		return err
	}
	req := &v1.SendTaskSuccessRequest{
		TaskToken: ctx.Runtime.data.TaskToken,
		Output:    string(result),
	}
	_, err = ctx.Client.GetClient().SendTaskSuccess(ctx.Context, req)
	if err != nil {
		return err
	}
	return nil
}

func (ctx *Context) SendTaskHeartbeat(message string) error {
	var err error
	req := &v1.SendTaskHeartbeatRequest{
		TaskToken: ctx.Runtime.data.TaskToken,
		Message:   message,
	}
	_, err = ctx.Client.GetClient().SendTaskHeartbeat(ctx.Context, req)
	if err != nil {
		return err
	}
	return nil
}

func (ctx *Context) UnmarshalInput(v interface{}) error {
	var err error
	data := ctx.Runtime.data.Input
	err = json.Unmarshal([]byte(data), v)
	if err != nil {
		return err
	}
	return nil
}
