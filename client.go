package skyflowclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
	defer resp.Body.Close()

	// 读取整个响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 首先尝试解析为自定义响应格式
	var customResp CustomResponse
	if err := json.Unmarshal(body, &customResp); err != nil {
		// 如果不是自定义格式，直接解析
		return json.Unmarshal(body, v)
	}

	// 检查业务错误
	if !customResp.Success {
		return fmt.Errorf("business error: code=%s, message=%s",
			customResp.ErrorCode, customResp.ErrorMessage)
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

	return json.Unmarshal(customResp.Data, v)
}

// CustomErrorDecoder 自定义错误解码器
func CustomErrorDecoder(ctx context.Context, resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var customResp CustomResponse
	if err := json.Unmarshal(body, &customResp); err != nil {
		// 如果不是自定义格式，使用默认错误处理
		return kratosHttp.DefaultErrorDecoder(ctx, resp)
	}

	if !customResp.Success {
		return fmt.Errorf("code=%s, message=%s, return_code=%d",
			customResp.ErrorCode, customResp.ErrorMessage, customResp.ReturnCode)
	}

	return nil
}
