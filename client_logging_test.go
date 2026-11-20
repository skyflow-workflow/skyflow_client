package skyflowclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	kratosErrors "github.com/go-kratos/kratos/v2/errors"
	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"
	apiV1 "github.com/skyflow-workflow/skyflow_backend/api/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

type fakeTaskHTTPClient struct {
	apiV1.SkyflowV1ServiceHTTPClient
	getActivityTask func(context.Context, *apiV1.GetActivityTaskRequest, ...kratosHttp.CallOption) (*apiV1.GetActivityTaskResponse, error)
	sendTaskSuccess func(context.Context, *apiV1.SendTaskSuccessRequest, ...kratosHttp.CallOption) (*emptypb.Empty, error)
	sendTaskFailure func(context.Context, *apiV1.SendTaskFailureRequest, ...kratosHttp.CallOption) (*emptypb.Empty, error)
}

func (client *fakeTaskHTTPClient) GetActivityTask(
	ctx context.Context,
	req *apiV1.GetActivityTaskRequest,
	opts ...kratosHttp.CallOption,
) (*apiV1.GetActivityTaskResponse, error) {
	if client.getActivityTask == nil {
		return nil, errors.New("getActivityTask is not configured")
	}
	return client.getActivityTask(ctx, req, opts...)
}

func (client *fakeTaskHTTPClient) SendTaskSuccess(
	ctx context.Context,
	req *apiV1.SendTaskSuccessRequest,
	opts ...kratosHttp.CallOption,
) (*emptypb.Empty, error) {
	if client.sendTaskSuccess == nil {
		return nil, errors.New("sendTaskSuccess is not configured")
	}
	return client.sendTaskSuccess(ctx, req, opts...)
}

func (client *fakeTaskHTTPClient) SendTaskFailure(
	ctx context.Context,
	req *apiV1.SendTaskFailureRequest,
	opts ...kratosHttp.CallOption,
) (*emptypb.Empty, error) {
	if client.sendTaskFailure == nil {
		return nil, errors.New("sendTaskFailure is not configured")
	}
	return client.sendTaskFailure(ctx, req, opts...)
}

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})
	return &output
}

func tokenHash(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])[:12]
}

func assertLogContains(t *testing.T, logs *bytes.Buffer, fragments ...string) {
	t.Helper()

	content := logs.String()
	for _, fragment := range fragments {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected logs to contain %q, got:\n%s", fragment, content)
		}
	}
}

func assertLogNotContains(t *testing.T, logs *bytes.Buffer, fragments ...string) {
	t.Helper()

	content := logs.String()
	for _, fragment := range fragments {
		if strings.Contains(content, fragment) {
			t.Fatalf("expected logs not to contain %q, got:\n%s", fragment, content)
		}
	}
}

func TestClient_GetActivityTaskOperationLogging(t *testing.T) {
	logs := captureLogs(t)
	const taskToken = "task-token-for-test"
	client := &Client{httpClient: &fakeTaskHTTPClient{
		getActivityTask: func(_ context.Context, _ *apiV1.GetActivityTaskRequest, _ ...kratosHttp.CallOption) (*apiV1.GetActivityTaskResponse, error) {
			return &apiV1.GetActivityTaskResponse{
				ActivityUri:      "activity:unittest/add",
				TaskToken:        taskToken,
				Input:            `{"x":1,"y":2}`,
				TimeoutSeconds:   10,
				HeartbeatSeconds: 5,
			}, nil
		},
	}}

	resp, err := client.GetActivityTask(context.Background(), &apiV1.GetActivityTaskRequest{
		ActivityUri:    "activity:unittest/add",
		TimeoutSeconds: 10,
	})
	if err != nil {
		t.Fatalf("GetActivityTask returned error: %v", err)
	}
	if resp.GetTaskToken() != taskToken {
		t.Fatalf("unexpected task token: %q", resp.GetTaskToken())
	}

	assertLogContains(t, logs,
		"operation=GetActivityTask",
		"activity_uri=activity:unittest/add",
		"task_found=true",
		"task_token_hash="+tokenHash(taskToken),
		"input_length=13",
		"heartbeat_seconds=5",
	)
	assertLogNotContains(t, logs, taskToken, `{"x":1,"y":2}`)
}

func TestClient_SendTaskSuccessOperationLogging(t *testing.T) {
	logs := captureLogs(t)
	const taskToken = "success-task-token"
	client := &Client{httpClient: &fakeTaskHTTPClient{
		sendTaskSuccess: func(_ context.Context, _ *apiV1.SendTaskSuccessRequest, _ ...kratosHttp.CallOption) (*emptypb.Empty, error) {
			return nil, nil
		},
	}}

	err := client.SendTaskSuccess(context.Background(), &apiV1.SendTaskSuccessRequest{
		TaskToken: taskToken,
		Output:    `{"result":3}`,
	})
	if err != nil {
		t.Fatalf("SendTaskSuccess returned error: %v", err)
	}

	assertLogContains(t, logs,
		"operation=SendTaskSuccess",
		"task_token_hash="+tokenHash(taskToken),
		"output_length=12",
	)
	assertLogNotContains(t, logs, taskToken, `{"result":3}`)
}

func TestClient_SendTaskFailureOperationLogging(t *testing.T) {
	logs := captureLogs(t)
	const taskToken = "failure-task-token"
	client := &Client{httpClient: &fakeTaskHTTPClient{
		sendTaskFailure: func(_ context.Context, _ *apiV1.SendTaskFailureRequest, _ ...kratosHttp.CallOption) (*emptypb.Empty, error) {
			return nil, nil
		},
	}}

	err := client.SendTaskFailure(context.Background(), &apiV1.SendTaskFailureRequest{
		TaskToken: taskToken,
		Error:     "SubError",
		Cause:     "sub result is negative",
	})
	if err != nil {
		t.Fatalf("SendTaskFailure returned error: %v", err)
	}

	assertLogContains(t, logs,
		"operation=SendTaskFailure",
		"task_token_hash="+tokenHash(taskToken),
		"task_error=SubError",
		"cause_length=22",
	)
	assertLogNotContains(t, logs, taskToken, "sub result is negative")
}

func TestClient_TaskOperationErrorLogging(t *testing.T) {
	logs := captureLogs(t)
	client := &Client{httpClient: &fakeTaskHTTPClient{
		sendTaskSuccess: func(_ context.Context, _ *apiV1.SendTaskSuccessRequest, _ ...kratosHttp.CallOption) (*emptypb.Empty, error) {
			return nil, errors.New("network unavailable")
		},
	}}

	err := client.SendTaskSuccess(context.Background(), &apiV1.SendTaskSuccessRequest{
		TaskToken: "error-task-token",
		Output:    "ok",
	})
	if err == nil {
		t.Fatal("expected SendTaskSuccess to return an error")
	}

	assertLogContains(t, logs,
		"operation=SendTaskSuccess",
		"skyflow client operation failed",
		`error="network unavailable"`,
	)
	assertLogNotContains(t, logs, "skyflow client operation succeeded")
}

func TestContext_SendTaskOperationLogging(t *testing.T) {
	logs := captureLogs(t)
	const taskToken = "context-task-token"
	client := &Client{httpClient: &fakeTaskHTTPClient{
		sendTaskSuccess: func(_ context.Context, _ *apiV1.SendTaskSuccessRequest, _ ...kratosHttp.CallOption) (*emptypb.Empty, error) {
			return nil, nil
		},
		sendTaskFailure: func(_ context.Context, _ *apiV1.SendTaskFailureRequest, _ ...kratosHttp.CallOption) (*emptypb.Empty, error) {
			return nil, nil
		},
	}}
	workerContext := NewContext(context.Background(), client, &ActivityTaskRuntime{
		Data: &apiV1.GetActivityTaskResponse{
			TaskToken: taskToken,
		},
	})

	if err := workerContext.SendTaskSuccess(map[string]int{"result": 3}); err != nil {
		t.Fatalf("SendTaskSuccess returned error: %v", err)
	}
	if err := workerContext.SendTaskFailure("TestError", "test cause"); err != nil {
		t.Fatalf("SendTaskFailure returned error: %v", err)
	}

	assertLogContains(t, logs,
		"operation=SendTaskSuccess",
		"operation=SendTaskFailure",
		"task_token_hash="+tokenHash(taskToken),
	)
	assertLogNotContains(t, logs, taskToken)
}

func TestActivityWorkerWithLoggerUpdatesExistingInstance(t *testing.T) {
	var output bytes.Buffer
	customLogger := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	client := &Client{httpClient: &fakeTaskHTTPClient{
		sendTaskSuccess: func(_ context.Context, _ *apiV1.SendTaskSuccessRequest, _ ...kratosHttp.CallOption) (*emptypb.Empty, error) {
			return nil, nil
		},
	}}
	worker, err := NewActivityWorker(client, 1)
	if err != nil {
		t.Fatalf("NewActivityWorker returned error: %v", err)
	}
	t.Cleanup(worker.workerPool.Release)

	activity := &WorkerActivity{
		Name: "custom-logger",
		Function: func(ctx *Context) error {
			return ctx.SendTaskSuccess("custom logger")
		},
	}
	if returned := worker.WithLogger(customLogger); returned != worker {
		t.Fatal("WithLogger should return the same ActivityWorker instance")
	}
	worker.runActivity(ActivityTaskRuntime{
		Activity: activity,
		Data: &apiV1.GetActivityTaskResponse{
			TaskToken: "custom-logger-token",
		},
	})

	defaultLogs := captureLogs(t)
	worker.WithLogger(nil)
	worker.runActivity(ActivityTaskRuntime{
		Activity: activity,
		Data: &apiV1.GetActivityTaskResponse{
			TaskToken: "default-logger-token",
		},
	})

	assertLogContains(t, &output,
		"operation=SendTaskSuccess",
		"task_token_hash="+tokenHash("custom-logger-token"),
	)
	assertLogNotContains(t, &output, "default-logger-token")
	assertLogContains(t, defaultLogs, "operation=SendTaskSuccess")
	assertLogNotContains(t, defaultLogs, "custom-logger-token")
}

func TestActivityWorkerWithIgnoreActivityTaskNotFound(t *testing.T) {
	logs := captureLogs(t)
	notFound := kratosErrors.New(
		int(apiV1.ErrorCode_ACTIVITY_TASK_NOT_FOUND),
		"ACTIVITY_TASK_NOT_FOUND",
		"activity task not found",
	)
	client := &Client{httpClient: &fakeTaskHTTPClient{
		getActivityTask: func(_ context.Context, _ *apiV1.GetActivityTaskRequest, _ ...kratosHttp.CallOption) (*apiV1.GetActivityTaskResponse, error) {
			return nil, notFound
		},
	}}
	worker, err := NewActivityWorker(client, 1)
	if err != nil {
		t.Fatalf("NewActivityWorker returned error: %v", err)
	}
	t.Cleanup(worker.workerPool.Release)

	if returned := worker.WithIgnoreActivityTaskNotFound(true); returned != worker {
		t.Fatal("WithIgnoreActivityTaskNotFound should return the same ActivityWorker instance")
	}
	if _, err := client.GetActivityTask(worker.taskContext(), &apiV1.GetActivityTaskRequest{
		ActivityUri: "activity:unittest/no-task",
	}); err == nil {
		t.Fatal("expected GetActivityTask to return ACTIVITY_TASK_NOT_FOUND")
	}

	assertLogContains(t, logs, "operation=GetActivityTask")
	assertLogNotContains(t, logs,
		"skyflow client operation failed",
		"ACTIVITY_TASK_NOT_FOUND",
		"activity task not found",
	)
}

func TestActivityWorkerLogsActivityTaskNotFoundByDefault(t *testing.T) {
	logs := captureLogs(t)
	notFound := kratosErrors.New(
		int(apiV1.ErrorCode_ACTIVITY_TASK_NOT_FOUND),
		"ACTIVITY_TASK_NOT_FOUND",
		"activity task not found",
	)
	client := &Client{httpClient: &fakeTaskHTTPClient{
		getActivityTask: func(_ context.Context, _ *apiV1.GetActivityTaskRequest, _ ...kratosHttp.CallOption) (*apiV1.GetActivityTaskResponse, error) {
			return nil, notFound
		},
	}}
	worker, err := NewActivityWorker(client, 1)
	if err != nil {
		t.Fatalf("NewActivityWorker returned error: %v", err)
	}
	t.Cleanup(worker.workerPool.Release)

	if _, err := client.GetActivityTask(worker.taskContext(), &apiV1.GetActivityTaskRequest{
		ActivityUri: "activity:unittest/no-task",
	}); err == nil {
		t.Fatal("expected GetActivityTask to return ACTIVITY_TASK_NOT_FOUND")
	}

	assertLogContains(t, logs,
		"skyflow client operation failed",
		"code = 13010",
		"reason = ACTIVITY_TASK_NOT_FOUND",
	)
}

func TestActivityWorkerIgnoreActivityTaskNotFoundOnlyAppliesToNotFound(t *testing.T) {
	logs := captureLogs(t)
	client := &Client{httpClient: &fakeTaskHTTPClient{
		getActivityTask: func(_ context.Context, _ *apiV1.GetActivityTaskRequest, _ ...kratosHttp.CallOption) (*apiV1.GetActivityTaskResponse, error) {
			return nil, errors.New("network unavailable")
		},
	}}
	worker, err := NewActivityWorker(client, 1)
	if err != nil {
		t.Fatalf("NewActivityWorker returned error: %v", err)
	}
	t.Cleanup(worker.workerPool.Release)
	worker.WithIgnoreActivityTaskNotFound(true)

	if _, err := client.GetActivityTask(worker.taskContext(), &apiV1.GetActivityTaskRequest{
		ActivityUri: "activity:unittest/network-error",
	}); err == nil {
		t.Fatal("expected GetActivityTask to return an error")
	}

	assertLogContains(t, logs,
		"skyflow client operation failed",
		`error="network unavailable"`,
	)
}

func TestIsActivityPollNotFound(t *testing.T) {
	activityNotFound := kratosErrors.New(
		int(apiV1.ErrorCode_ACTIVITY_NOT_FOUND),
		"ACTIVITY_NOT_FOUND",
		"activity not found",
	)
	activityTaskNotFound := kratosErrors.New(
		int(apiV1.ErrorCode_ACTIVITY_TASK_NOT_FOUND),
		"ACTIVITY_TASK_NOT_FOUND",
		"activity task not found",
	)

	if !isActivityPollNotFound(activityNotFound) {
		t.Fatal("expected ACTIVITY_NOT_FOUND to be treated as a normal poll result")
	}
	if !isActivityPollNotFound(activityTaskNotFound) {
		t.Fatal("expected ACTIVITY_TASK_NOT_FOUND to be treated as a normal poll result")
	}
	if isActivityPollNotFound(errors.New("network unavailable")) {
		t.Fatal("expected non-not-found errors to use the error backoff interval")
	}
}

func TestCustomResponseDecoderUsesProvidedLogger(t *testing.T) {
	var output bytes.Buffer
	customLogger := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	body := `{"success":true,"data":{"value":"ok"}}`
	var result struct {
		Value string `json:"value"`
	}

	err := CustomResponseDecoder(
		withLoggerContext(context.Background(), customLogger),
		&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
		},
		&result,
	)
	if err != nil {
		t.Fatalf("CustomResponseDecoder returned error: %v", err)
	}
	if result.Value != "ok" {
		t.Fatalf("unexpected result: %+v", result)
	}

	assertLogContains(t, &output,
		"skyflow client response received",
		"status_code=200",
		"body_length=38",
	)
	assertLogNotContains(t, &output, body)
}
