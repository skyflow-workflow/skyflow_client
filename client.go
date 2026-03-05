package skyflowclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-kratos/kratos/v2/errors"
	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"
	apiV1 "github.com/skyflow-workflow/skyflow_backend/api/v1"
)

type Client struct {
	httpClient apiV1.SkyflowV1ServiceHTTPClient
}

func NewClient(address string) (*Client, error) {

	httpClient, err := kratosHttp.NewClient(
		context.Background(),
		kratosHttp.WithEndpoint(address),
		kratosHttp.WithErrorDecoder(CustomErrorDecoder),
		kratosHttp.WithResponseDecoder(CustomResponseDecoder),
	)
	if err != nil {
		return nil, err
	}

	client := apiV1.NewSkyflowV1ServiceHTTPClient(
		httpClient,
	)
	return &Client{
		httpClient: client,
	}, nil
}

func (client *Client) Close() error {
	return nil
}

func (client *Client) GetClient() apiV1.SkyflowV1ServiceHTTPClient {
	return client.httpClient
}

func (client *Client) GetActivityTask(ctx context.Context, req *apiV1.GetActivityTaskRequest) (*apiV1.GetActivityTaskResponse, error) {
	return client.httpClient.GetActivityTask(ctx, req)
}

func (client *Client) SendTaskSuccess(ctx context.Context, req *apiV1.SendTaskSuccessRequest) error {
	_, err := client.httpClient.SendTaskSuccess(ctx, req)
	return err
}

func (client *Client) SendTaskFailure(ctx context.Context, req *apiV1.SendTaskFailureRequest) error {
	_, err := client.httpClient.SendTaskFailure(ctx, req)
	return err
}

func (client *Client) SendTaskHeartbeat(ctx context.Context, req *apiV1.SendTaskHeartbeatRequest) error {
	_, err := client.httpClient.SendTaskHeartbeat(ctx, req)
	return err
}

func (client *Client) CreateOrUpdateActivity(ctx context.Context, req *apiV1.CreateActivityRequest,
) (*apiV1.CreateActivityResponse, error) {
	return client.httpClient.CreateOrUpdateActivity(ctx, req)
}

func (client *Client) CreateOrUpdateNamespace(ctx context.Context, req *apiV1.CreateNamespaceRequest,
) (*apiV1.CreateNamespaceResponse, error) {
	resp, err := client.httpClient.CreateOrUpdateNamespace(ctx, req)
	return resp, err
}

func (client *Client) CreateOrUpdateStateMachine(ctx context.Context, req *apiV1.CreateStateMachineRequest,
) (*apiV1.CreateStateMachineResponse, error) {
	return client.httpClient.CreateOrUpdateStateMachine(ctx, req)
}

func (client *Client) StartExecution(ctx context.Context, req StartExecutionRequest,
) (*apiV1.StartExecutionResponse, error) {

	inputByte, err := json.Marshal(req.Input)
	if err != nil {
		return nil, err
	}
	inputStr := string(inputByte)

	pbReq := &apiV1.StartExecutionRequest{
		StatemachineUri: req.StatemachineUri,
		Input:           inputStr,
		ExecutionName:   req.ExecutionName,
		Definition:      req.Definition,
		Title:           req.Title,
	}
	return client.httpClient.StartExecution(ctx, pbReq)

}

// CustomResponse 对应服务端的 Response 结构
type CustomResponse struct {
	Success      bool            `json:"success"`
	ErrorCode    string          `json:"error_code"`
	ReturnCode   int             `json:"return_code"`
	ErrorMessage string          `json:"error_message"`
	Data         json.RawMessage `json:"data"`
}

// CustomResponseDecoder 自定义响应解码器
func CustomResponseDecoder(ctx context.Context, resp *http.Response, v any) error {

	// 读取响应体之前先检查是否为空
	if resp.Body == nil {
		return errors.New(500, "EMPTY_RESPONSE", "Empty response body")
	}
	// 读取整个响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	respBodyStr := string(bodyBytes)

	slog.Info(respBodyStr)

	// 将读取的内容重新设置回 Body，以便后续处理
	// 关键步骤：重置 Body
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// 首先尝试解析为自定义响应格式
	var customResp CustomResponse
	err = json.Unmarshal(bodyBytes, &customResp)
	if err != nil {
		// 如果不是自定义格式，直接解析
		return json.Unmarshal(bodyBytes, v)
	}

	// 检查业务错误
	if !customResp.Success {
		kratosErr := errors.New(customResp.ReturnCode, customResp.ErrorCode, customResp.ErrorMessage)
		return kratosErr
	}

	// 如果 v 是 *http.Response，直接返回原始响应
	if response, ok := v.(*http.Response); ok {
		*response = *resp
		return nil
	}

	// 如果不需要解析数据
	if v == nil {
		return nil
	}

	// 解析 data 字段
	if len(customResp.Data) == 0 {
		return nil
	}

	err = json.Unmarshal(customResp.Data, v)
	if err != nil {
		return err
	}
	return nil
}

// CustomErrorDecoder 自定义错误解码器
func CustomErrorDecoder(ctx context.Context, resp *http.Response) error {
	if resp.Body == nil {
		return errors.New(500, "EMPTY_RESPONSE", "Empty response body")
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	// 关键步骤：重置 Body
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var customResp CustomResponse
	err = json.Unmarshal(bodyBytes, &customResp)
	if err != nil {
		// 如果不是自定义格式，使用默认错误处理
		return kratosHttp.DefaultErrorDecoder(ctx, resp)
	}

	if customResp.ReturnCode == 0 || customResp.Success {
		return nil
	}
	kratosErr := errors.New(customResp.ReturnCode, customResp.ErrorCode, customResp.ErrorMessage)
	return kratosErr
}
