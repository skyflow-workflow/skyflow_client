package skyflowclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"
	apiV1 "github.com/skyflow-workflow/skyflow_backend/api/v1"
)

const defaultHTTPTimeout = 30 * time.Second

type Client struct {
	httpClient      apiV1.SkyflowV1ServiceHTTPClient
	transportClient *kratosHttp.Client
}

type clientOptions struct {
	httpTimeout time.Duration
}

// ClientOption configures a Client.
type ClientOption func(*clientOptions)

// WithHTTPTimeout sets the underlying Kratos HTTP client timeout.
func WithHTTPTimeout(timeout time.Duration) ClientOption {
	return func(options *clientOptions) {
		options.httpTimeout = timeout
	}
}

// NewClient creates a Client. SDK logs made directly through Client use the
// current slog.Default() logger. ActivityWorker injects its configured logger
// through the request context.
func NewClient(address string, opts ...ClientOption) (*Client, error) {
	options := clientOptions{httpTimeout: defaultHTTPTimeout}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	if options.httpTimeout <= 0 {
		return nil, fmt.Errorf("invalid HTTP timeout: %s", options.httpTimeout)
	}
	httpClient, err := kratosHttp.NewClient(
		context.Background(),
		kratosHttp.WithEndpoint(address),
		kratosHttp.WithTimeout(options.httpTimeout),
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
		httpClient:      client,
		transportClient: httpClient,
	}, nil
}

func (client *Client) Close() error {
	if client.transportClient == nil {
		return nil
	}
	return client.transportClient.Close()
}

func (client *Client) GetClient() apiV1.SkyflowV1ServiceHTTPClient {
	return client.httpClient
}

func (client *Client) GetActivityTask(ctx context.Context, req *apiV1.GetActivityTaskRequest) (*apiV1.GetActivityTaskResponse, error) {
	startedAt := time.Now()
	logger := loggerFromContext(ctx)
	logger.InfoContext(ctx, "skyflow client operation started",
		"operation", "GetActivityTask",
		"activity_uri", req.GetActivityUri(),
		"timeout_seconds", req.GetTimeoutSeconds(),
	)

	resp, err := client.httpClient.GetActivityTask(ctx, req)
	if err != nil {
		if shouldIgnoreActivityTaskNotFound(ctx, err) {
			return nil, err
		}
		logger.ErrorContext(ctx, "skyflow client operation failed",
			"operation", "GetActivityTask",
			"activity_uri", req.GetActivityUri(),
			"timeout_seconds", req.GetTimeoutSeconds(),
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"error", err.Error(),
		)
		return nil, err
	}

	logger.InfoContext(ctx, "skyflow client operation succeeded",
		"operation", "GetActivityTask",
		"activity_uri", resp.GetActivityUri(),
		"task_found", resp.GetTaskToken() != "",
		"task_token_hash", taskTokenHash(resp.GetTaskToken()),
		"input_length", len(resp.GetInput()),
		"timeout_seconds", resp.GetTimeoutSeconds(),
		"heartbeat_seconds", resp.GetHeartbeatSeconds(),
		"duration_ms", time.Since(startedAt).Milliseconds(),
	)
	return resp, nil
}

func (client *Client) SendTaskSuccess(ctx context.Context, req *apiV1.SendTaskSuccessRequest) error {
	startedAt := time.Now()
	logger := loggerFromContext(ctx)
	logger.InfoContext(ctx, "skyflow client operation started",
		"operation", "SendTaskSuccess",
		"task_token_hash", taskTokenHash(req.GetTaskToken()),
		"output_length", len(req.GetOutput()),
	)

	_, err := client.httpClient.SendTaskSuccess(ctx, req)
	if err != nil {
		logger.ErrorContext(ctx, "skyflow client operation failed",
			"operation", "SendTaskSuccess",
			"task_token_hash", taskTokenHash(req.GetTaskToken()),
			"output_length", len(req.GetOutput()),
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"error", err.Error(),
		)
		return err
	}
	logger.InfoContext(ctx, "skyflow client operation succeeded",
		"operation", "SendTaskSuccess",
		"task_token_hash", taskTokenHash(req.GetTaskToken()),
		"output_length", len(req.GetOutput()),
		"duration_ms", time.Since(startedAt).Milliseconds(),
	)
	return err
}

func (client *Client) SendTaskFailure(ctx context.Context, req *apiV1.SendTaskFailureRequest) error {
	startedAt := time.Now()
	logger := loggerFromContext(ctx)
	logger.InfoContext(ctx, "skyflow client operation started",
		"operation", "SendTaskFailure",
		"task_token_hash", taskTokenHash(req.GetTaskToken()),
		"task_error", req.GetError(),
		"cause_length", len(req.GetCause()),
	)

	_, err := client.httpClient.SendTaskFailure(ctx, req)
	if err != nil {
		logger.ErrorContext(ctx, "skyflow client operation failed",
			"operation", "SendTaskFailure",
			"task_token_hash", taskTokenHash(req.GetTaskToken()),
			"task_error", req.GetError(),
			"cause_length", len(req.GetCause()),
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"error", err.Error(),
		)
		return err
	}
	logger.InfoContext(ctx, "skyflow client operation succeeded",
		"operation", "SendTaskFailure",
		"task_token_hash", taskTokenHash(req.GetTaskToken()),
		"task_error", req.GetError(),
		"cause_length", len(req.GetCause()),
		"duration_ms", time.Since(startedAt).Milliseconds(),
	)
	return err
}

func taskTokenHash(token string) string {
	if token == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])[:12]
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
	loggerFromContext(ctx).DebugContext(ctx, "skyflow client response received",
		"status_code", resp.StatusCode,
		"body_length", len(bodyBytes),
	)

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

type loggerContextKey struct{}

type ignoreActivityTaskNotFoundContextKey struct{}

func withLoggerContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

func loggerFromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}
	if logger, ok := ctx.Value(loggerContextKey{}).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return slog.Default()
}

func withIgnoreActivityTaskNotFound(ctx context.Context, ignore bool) context.Context {
	return context.WithValue(ctx, ignoreActivityTaskNotFoundContextKey{}, ignore)
}

func shouldIgnoreActivityTaskNotFound(ctx context.Context, err error) bool {
	if ctx == nil || err == nil {
		return false
	}
	ignore, ok := ctx.Value(ignoreActivityTaskNotFoundContextKey{}).(bool)
	if !ok || !ignore {
		return false
	}
	return errors.FromError(err).Code == int32(apiV1.ErrorCode_ACTIVITY_TASK_NOT_FOUND)
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
