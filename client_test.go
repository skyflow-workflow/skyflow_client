package skyflowclient

import (
	"fmt"
	"testing"
)

var sf_obj *SkyFlowClient

func init() {
	var err error
	server := "http://9.134.6.17:8080/"
	// server := "9.134.6.17:8080"
	sf_obj, err = NewSkyFlowClient(server)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(sf_obj)
}

func TestSkyFlow(t *testing.T) {

	data, err := sf_obj.ListRepositories()
	if err != nil {
		fmt.Println("error")
		fmt.Println(err.Error())
		return
	}
	fmt.Println(data)
	for _, re := range data {
		fmt.Println(re.Name, re.Comment)
		acts, err := sf_obj.ListActivities(re.Name)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		for _, act := range acts {
			fmt.Println(act.Name)
			actinfo, err := sf_obj.DescribeActivity(act.URI)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			fmt.Println(actinfo)
		}
	}

}

func TestDescribeActivity(t *testing.T) {

	var repoName = "qyweixin"

	acts, err := sf_obj.ListActivities(repoName)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	for _, act := range acts {
		fmt.Println(act.Name)
		actinfo, err := sf_obj.DescribeActivity(act.URI)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Println(actinfo)
	}
}
func TestDescribeExecution(t *testing.T) {

	var uuid = "669ef589-732a-11ea-bf77-0242ac110007"

	exe, err := sf_obj.DescribeExecution(uuid)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(exe)
}
func TestListExecutions(t *testing.T) {

	var cond = ListExecutionCondition{
		PageNumber: 50,
	}

	exes, err := sf_obj.ListExecutions(cond)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("exeuction count :", len(exes))

	for _, exe := range exes {
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Println(exe)
	}
}

func TestGetExecutionEvents(t *testing.T) {
	uuid := "8301d1dd-0adf-11eb-ad1d-52540002d945"
	events, err := sf_obj.GetExecutionHistory(uuid)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(events)
}
