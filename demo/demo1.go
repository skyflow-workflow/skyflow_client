package main

import (
	"context"
	"log/slog"
	"time"

	skyflowclient "github.com/skyflow-workflow/skyflow_client"
)

func main() {

	var err error
	// address := "http://127.0.0.1:8080/"
	address := "http://t1-skyflow.shizhuang-inc.net/"
	client, err := skyflowclient.NewClient(address)
	if err != nil {
		slog.Error("NewClient error", "error", err)
		return
	}
	namespaces := []*skyflowclient.WorkerNamespace{
		skyflowclient.NewWorkerNamespace(
			"unittest",
			"",
			[]string{"demo"},
			skyflowclient.NewWorkerActivity("add", addActivity, addActivityDoc),
			skyflowclient.NewWorkerActivity("sub", subActivity, subActivityDoc),
		),
	}
	aw, err := skyflowclient.NewActivityWorker(client, 1000,
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

	// err = create_demo_execution(client)
	// if err != nil {
	// 	slog.Error("create_demo_execution error", "error", err)
	// 	return
	// }
	// 启动worker
	aw.Run()
	// 等10s钟
	time.Sleep(10 * time.Second)
	select {}
	// 停止worker
	aw.Stop()
}

func create_demo_execution(client *skyflowclient.Client) error {
	var err error
	statemachineUri := "statemachine:unittest/task_with_2_step"
	input := map[string]int{
		"a": 1,
		"b": 3,
	}

	Execution, err := client.StartExecution(
		context.Background(),
		skyflowclient.StartExecutionRequest{
			StatemachineUri: statemachineUri,
			Title:           "test go worker",
			Input:           input,
		})
	if err != nil {
		slog.Error("StartExecution error", "error", err)
		return err
	}
	slog.Info("StartExecution success", "Execution", Execution)
	return nil
}

var addActivityDoc = `
add 计算两个数的和
input:
	{
		"x" : 1,
		"y" : 2
	}
output:
	3

`

func addActivity(ctx *skyflowclient.Context) error {
	var err error

	type InputFormat struct {
		X int `json:"x"`
		Y int `json:"y"`
	}

	var inputdata = InputFormat{
		X: 0,
	}
	err = ctx.UnmarshalInput(&inputdata)
	if err != nil {
		return err
	}

	sum := inputdata.X + inputdata.Y

	slog.Info("addActivity", "sum", sum)
	err = ctx.SendTaskSuccess(sum)
	return err
}

var subActivityDoc = `
sub 计算两个数的差
input:
	{
		"x" : 1,
		"y" : 2
	}
output:
	{
		"result" : 1
	}

error:
	SubError: 差小于0
`

func subActivity(ctx *skyflowclient.Context) error {
	var err error

	type InputFormat struct {
		X int `json:"x"`
		Y int `json:"y"`
	}

	type OutputFormat struct {
		Result int `json:"result"`
	}

	var inputdata = InputFormat{
		X: 0,
		Y: 10,
	}
	err = ctx.UnmarshalInput(&inputdata)
	if err != nil {
		return err
	}

	result := inputdata.X - inputdata.Y
	slog.Info("subActivity", "result", result)

	if result < 0 {
		err = ctx.SendTaskFailure("SubError", "sub is not allown")
		return err
	}

	var output = OutputFormat{
		Result: result,
	}

	err = ctx.SendTaskSuccess(output)
	return err
}
