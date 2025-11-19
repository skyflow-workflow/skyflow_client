package main

import (
	"fmt"
	"time"

	skyflowclient "github.com/skyflow-workflow/skyflow_client"
)

func main() {

	var err error
	address := "http://127.0.0.1:8080/"
	client, err := skyflowclient.NewSkyFlowClient(address)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	repos := []*skyflowclient.WorkerRepository{
		skyflowclient.NewWorkerRepository("testrepo", []string{"demo/testact3.json"},
			skyflowclient.NewWorkerActivity("add", addActivity, addActivityDoc),
			skyflowclient.NewWorkerActivity("checkresult", checkresult, checkresultDoc),
		),
	}
	aw, err := skyflowclient.NewActivityWorker(client, 1000)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	err = aw.Register()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	smuri := "statemachine:testrepo/testact"
	input := map[string]int{
		"x": 1,
		"y": 3,
	}
	apinewexe, err := client.StartExecution(smuri, "test go worker", input)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(apinewexe)
	// 启动worker
	aw.Run()
	// 等10s钟
	time.Sleep(10 * time.Second)
	// 停止worker
	aw.Stop()
}

var addActivityDoc = `
add 计算两个数的和
输入:
	{
		"x" : 1,
		"y" : 2
	}
输出:

3

`

func addActivity(input string, token string, sf *skyflowclient.SkyFlowClient) error {
	var err error
	fmt.Println(input, token)

	type InputFormat struct {
		X int `json:"x" post:"required"`
		Y int `json:"y" post:"required notzero"`
	}

	var inputdata = InputFormat{
		X: 0,
	}
	err = skyflowclient.UnmarshalInput(input, &inputdata)
	if err != nil {
		return err
	}

	sum := inputdata.X + inputdata.Y

	err = sf.SendTaskSuccess(token, sum)
	return err
}

var checkresultDoc = `dfe
check input
输入:
输出:
`

func checkresult(input string, token string, sf *skyflowclient.SkyFlowClient) error {
	var err error

	fmt.Println(input, token)

	err = sf.SendTaskSuccess(token, input)
	return err
}
