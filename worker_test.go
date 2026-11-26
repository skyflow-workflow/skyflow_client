package skyflowclient

import (
	"log/slog"
	"testing"
)

func TestWorker(t *testing.T) {

	// w := Worker{}
	// w.AddRepository()

}

func TestWorker_Register(t *testing.T) {
	requireSkyflowServer(t)

	client, err := NewClient(skyflowTestAddress())
	if err != nil {
		slog.Error("NewClient error", "error", err)
		return
	}

	namespaces := []*WorkerNamespace{
		NewWorkerNamespace(
			"unittest",
			"",
			// []string{"demo/testact3.json"},
			[]string{},
			NewWorkerActivity("add", FunctionAdd, "add 计算两个数的和"),
		),
	}
	aw, err := NewActivityWorker(client, 1000,
		namespaces...)
	if err != nil {
		slog.Error("NewActivityWorker error", "error", err)
		return
	}
	err = aw.Register()
	if err != nil {
		slog.Error("Register error", "error", err)
		return
	}

}

func FunctionAdd(ctx *Context) error {

	type InputFormat struct {
		X int `json:"x"`
		Y int `json:"y"`
	}

	var inputdata = InputFormat{
		X: 0,
	}
	err := ctx.UnmarshalInput(&inputdata)
	if err != nil {
		return err
	}
	sum := inputdata.X + inputdata.Y
	err = ctx.SendTaskSuccess(sum)
	return err
}
