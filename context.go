package skyflowclient

type ActivityRuntime struct {
}

// Context Skyflow Worker Context
// Context 上下文信息，用于传递给Activity 的参数
type Context struct {
	Client *Client
	Worker *ActivityWorker
}

// NewContext create a new context
func NewContext(client *Client, worker *ActivityWorker) *Context {
	return &Context{
		Client: client,
		Worker: worker,
	}
}

func (ctx *Context) SendTaskFailure(errorname string, cause string) error {
	var err error
	err = ctx.Client.client.SendTaskFailure(ctx.Worker.skyflow.ctxToken, errorname, cause)
	if err != nil {
		return err
	}
	return nil
}

func (ctx *Context) SendTaskSuccess(result map[string]interface{}) error {
	var err error
	err = ctx.Client.client.SendTaskSuccess(ctx.Worker.skyflow.ctxToken, result)
	if err != nil {
		return err
	}
	return nil
}
