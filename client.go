package skyflowclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SkyFlowError Skyflow error message
type SkyFlowError struct {
	ErrorCode    string
	ErrorMessage string
}

// SkyFlowErrorCode skyflow 错误码，方便做处理
var SkyFlowErrorCode = struct {
	ActivityTaskNotFound string
}{
	ActivityTaskNotFound: "ActivityTaskNotFound", //activity 未找到
}

// Error  for error type interface method
func (sfe SkyFlowError) Error() string {
	msg := fmt.Sprintf("ErrorCode: [ %s ] ErrorMessage: [ %s ]", sfe.ErrorCode, sfe.ErrorMessage)
	return msg
}

// SkyFlowClient skyflow client
type SkyFlowClient struct {
	server  string
	version string
	url     *url.URL
}

// NewSkyFlowClient  create skyflow client object
func NewSkyFlowClient(address string) (*SkyFlowClient, error) {

	var err error
	base, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	sf := SkyFlowClient{
		server:  address,
		version: "v1",
		url:     base,
	}

	return &sf, nil
}

func (sf *SkyFlowClient) httpproxy(apiname string, data interface{}, retval interface{}) error {
	var err error
	var resp *http.Response
	relurl := fmt.Sprintf("api/%s/%s", sf.version, apiname)

	u, err := url.Parse(relurl)
	if err != nil {
		return err
	}
	queryURL := sf.url.ResolveReference(u).String()

	queryData := ""
	if data != nil {
		queryDataByte, err := json.Marshal(data)
		if err != nil {
			return err
		}
		queryData = string(queryDataByte)
	}
	resp, err = http.Post(queryURL, "", strings.NewReader(queryData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	rm := ResponseMessage{}
	err = json.Unmarshal(body, &rm)
	if err != nil {
		return err
	}
	if !rm.Success {
		sfe := SkyFlowError{
			ErrorCode:    rm.ErrorCode,
			ErrorMessage: rm.ErrorMessage,
		}
		return sfe
	}
	err = json.Unmarshal(body, retval)

	return err
}

// GetActivityTask 查询待执行的活动
func (sf *SkyFlowClient) GetActivityTask(uri string) (APIActivityTask, error) {
	var err error
	apiname := "GetActivityTask"

	var queryDict = map[string]string{
		"activityUri": uri,
	}
	// APIResponseMessage  API返回值结构
	type APIResponseMessage struct {
		Data APIActivityTask `json:"Data"`
	}

	result := APIResponseMessage{}

	err = sf.httpproxy(apiname, queryDict, &result)
	return result.Data, err

}

// CreateRepository Create Repository
// if repository exist , update it  , if not exist , create it
func (sf *SkyFlowClient) CreateRepository(name string) error {

	var err error
	apiname := "CreateRepository"
	var queryDict = map[string]string{
		"repository": name,
	}
	// APIResponseMessage  API返回值结构
	type APIResponseMessage struct {
		Data APIRepository `json:"Data"`
	}

	result := APIResponseMessage{}

	err = sf.httpproxy(apiname, queryDict, &result)
	return err

}

// ListStateMachines 获得一个仓库下的所有状态机列表
/* repository :仓库名
 */
func (sf *SkyFlowClient) ListStateMachines(repository string) ([]APIStateMachine, error) {
	var err error
	apiname := "ListStateMachines"

	var queryDict = map[string]string{
		"repository": repository,
	}
	// APIResponseMessage  API返回值结构
	type APIResponseMessage struct {
		Data []APIStateMachine `json:"Data"`
	}

	result := APIResponseMessage{}
	err = sf.httpproxy(apiname, queryDict, &result)
	return result.Data, err

}

// DescribeStateMachine 获得一个StateMachine的信息
func (sf *SkyFlowClient) DescribeStateMachine(activityURI string) (APIStateMachine, error) {
	var err error
	apiname := "DescribeStateMachine"

	var queryDict = map[string]string{
		"activityUri": activityURI,
	}
	// APIResponseMessage  API返回值结构
	type APIResponseMessage struct {
		Data APIStateMachine `json:"Data"`
	}

	result := APIResponseMessage{}
	err = sf.httpproxy(apiname, queryDict, &result)
	return result.Data, err
}

// CreateActivity  创建一个活动
// @repository  仓库名
// @name  活动名称
// comment 活动说明
func (sf *SkyFlowClient) CreateActivity(repository string, name string, comment string) (APIActivity, error) {
	var err error
	apiname := "CreateActivity"

	var queryDict = map[string]string{
		"repository": repository,
		"name":       name,
		"comment":    comment,
	}
	// APIResponseMessage  API返回值结构
	type APIResponseMessage struct {
		Data APIActivity `json:"Data"`
	}

	result := APIResponseMessage{}
	err = sf.httpproxy(apiname, queryDict, &result)
	return result.Data, err
}

// DescribeActivity 获得一个Activity的信息
func (sf *SkyFlowClient) DescribeActivity(activityURI string) (APIActivity, error) {
	var err error
	apiname := "DescribeActivity"

	var queryDict = map[string]string{
		"activityUri": activityURI,
	}
	// APIResponseMessage  API返回值结构
	type APIResponseMessage struct {
		Data APIActivity `json:"Data"`
	}

	result := APIResponseMessage{}
	err = sf.httpproxy(apiname, queryDict, &result)
	return result.Data, err
}

// ListActivities 获得一个仓库下的所有活动列表
/* repository :仓库名
 */
func (sf *SkyFlowClient) ListActivities(repository string) ([]APIActivity, error) {
	var err error
	apiname := "ListActivities"

	var queryDict = map[string]string{
		"repository": repository,
	}
	// APIResponseMessage  API返回值结构
	type APIResponseMessage struct {
		Data []APIActivity `json:"Data"`
	}

	result := APIResponseMessage{}
	err = sf.httpproxy(apiname, queryDict, &result)
	return result.Data, err
}

// ListRepositories  list repository information
func (sf *SkyFlowClient) ListRepositories() ([]APIRepository, error) {
	var err error
	apiname := "ListRepositories"
	// APIResponseMessage  API返回值结构
	type APIResponseMessage struct {
		Data []APIRepository `json:"Data"`
	}

	result := APIResponseMessage{}
	err = sf.httpproxy(apiname, nil, &result)
	return result.Data, err
}

// CreateStateMachine  创建一个状态机
// @repository  仓库名
// @name  状态机名称
// @content  状态机内容
// @comment 状态机说明
func (sf *SkyFlowClient) CreateStateMachine(repository string,
	name string, definition string, comment string) (APIStateMachine, error) {
	var err error
	apiname := "CreateStateMachine"

	var queryDict = map[string]string{
		"repository": repository,
		"name":       name,
		"definition": definition,
		"comment":    comment,
	}
	// APIResponseMessage  API返回值结构
	type APIResponseMessage struct {
		Data APIStateMachine `json:"Data"`
	}

	result := APIResponseMessage{}
	err = sf.httpproxy(apiname, queryDict, &result)
	return result.Data, err
}

// StartExecution 创建一个新的 Execution
/*  参数
 *  stateMachineUri 状态机URI
 *  title Execution 标题
 *  input Execution 初始输入
 */
func (sf *SkyFlowClient) StartExecution(stateMachineURI string, title string, input interface{}) (APINewExecution, error) {
	var err error
	apiname := "StartExecution"

	var inputStr string

	if input == nil {
		inputStr = ""
	} else {
		inputByte, err := json.Marshal(input)
		if err != nil {
			return APINewExecution{}, err
		}
		inputStr = string(inputByte)
	}

	var queryDict = map[string]string{
		"stateMachineUri": stateMachineURI,
		"input":           inputStr,
		"title":           title,
	}
	// APIResponseMessage  API返回值结构
	type APIResponseMessage struct {
		Data APINewExecution `json:"Data"`
	}

	result := APIResponseMessage{}
	err = sf.httpproxy(apiname, queryDict, &result)
	return result.Data, err
}

// StopExecution 停止一个Execution
func (sf *SkyFlowClient) StopExecution(uuid string, errorcode string, cause string) error {
	var err error
	apiname := "StopExecution"

	var queryDict = map[string]string{
		"uuid":  uuid,      //任务ID,  必须
		"error": errorcode, //错误类型 , 非必须
		"cause": cause,     // 错误原因说明， 非必须
	}

	//Result API返回值类型， 返回停止的时间
	type Result struct {
		FinishTime time.Time `json:"FinishTime"`
	}
	// APIResponseMessage  API返回值结构
	type APIResponseMessage struct {
		Data Result `json:"Data"`
	}
	result := APIResponseMessage{}
	err = sf.httpproxy(apiname, queryDict, &result)
	return err
}

// DescribeExecution 获得一个Execution的信息
func (sf *SkyFlowClient) DescribeExecution(uuid string) (APIExecution, error) {
	var err error
	apiname := "DescribeExecution"

	var queryDict = map[string]string{
		"UUID": uuid,
	}
	// APIResponseMessage  API返回值结构
	type APIResponseMessage struct {
		Data APIExecution `json:"Data"`
	}

	result := APIResponseMessage{}
	err = sf.httpproxy(apiname, queryDict, &result)
	return result.Data, err
}

// ListExecutions  list execution information
func (sf *SkyFlowClient) ListExecutions(cond ListExecutionCondition) ([]APIExecution, error) {
	var err error
	apiname := "ListExecutions"
	// APIResponseMessage  API返回值结构
	type APIResponseMessage struct {
		Data []APIExecution `json:"Data"`
	}
	result := APIResponseMessage{}
	err = sf.httpproxy(apiname, cond, &result)
	return result.Data, err
}

// SendTaskSuccess 发送任务成功执行的结果
// @taskToken  task 执行的token
// @output  task 执行的输出
func (sf *SkyFlowClient) SendTaskSuccess(taskToken string, output interface{}) error {
	var err error
	apiname := "SendTaskSuccess"
	outputbyte, err := json.Marshal(output)
	if err != nil {
		return err
	}
	var queryDict = map[string]string{
		"taskToken": taskToken,
		"output":    string(outputbyte),
	}
	// APIResponseMessage  API返回值结构
	type APIResponseMessage struct {
	}
	result := APIResponseMessage{}
	err = sf.httpproxy(apiname, queryDict, &result)
	return err
}

// SendTaskFailure  发送任务执行失败的原因
// @taskToken  task 执行的token
// @error  task 执行的token
// @cause  task 执行的输出
func (sf *SkyFlowClient) SendTaskFailure(taskToken string, errorcode string, cause string) error {
	var err error
	apiname := "SendTaskFailure"
	var queryDict = map[string]string{
		"taskToken": taskToken,
		"error":     errorcode,
		"cause":     cause,
	}
	// APIResponseMessage  API返回值结构
	type APIResponseMessage struct {
	}
	result := APIResponseMessage{}
	err = sf.httpproxy(apiname, queryDict, &result)
	return err
}

// SendTaskHeartbeat  发送任务执行心跳信息
// @taskToken task执行的token
// @message   心跳附加的日志信息
func (sf *SkyFlowClient) SendTaskHeartbeat(taskToken string, message string) error {
	var err error
	apiname := "SendTaskHeartbeat"
	var queryDict = map[string]string{
		"taskToken": taskToken,
		"message":   message,
	}
	// APIResponseMessage  API返回值结构
	type APIResponseMessage struct {
	}
	result := APIResponseMessage{}
	err = sf.httpproxy(apiname, queryDict, &result)
	return err
}

// GetExecutionHistory  获得Execution的执行日志
// @uuid Execution UUID
func (sf *SkyFlowClient) GetExecutionHistory(uuid string) ([]APIExecutionEvent, error) {
	var err error
	apiname := "GetExecutionHistory"
	var queryDict = map[string]string{
		"executionID": uuid,
	}
	// APIResponseMessage  API返回值结构
	type APIResponseMessage struct {
		Data []APIExecutionEvent `json:"Data"`
	}
	result := APIResponseMessage{}
	err = sf.httpproxy(apiname, queryDict, &result)
	return result.Data, err
}
