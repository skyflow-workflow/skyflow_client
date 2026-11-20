package skyflowclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-playground/assert/v2"
	apiV1 "github.com/skyflow-workflow/skyflow_backend/api/v1"
)

func TestClient_GetActivityTaskHonorsHTTPTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		time.Sleep(2100 * time.Millisecond)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"error_code":"","return_code":0,"error_message":"","data":{"activity_uri":"activity:unittest/panic","task_token":"1234567890","input":"{}","timeout_seconds":0,"heartbeat_seconds":0}}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, WithHTTPTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	resp, err := client.GetActivityTask(context.Background(), &apiV1.GetActivityTaskRequest{
		ActivityUri: "activity:unittest/panic",
	})
	if err != nil {
		t.Fatalf("GetActivityTask after 2.1s: %v", err)
	}
	if resp.GetTaskToken() != "1234567890" {
		t.Fatalf("task token = %q", resp.GetTaskToken())
	}
}

func TestNewClient_WithHTTPTimeout(t *testing.T) {
	client, err := NewClient("http://127.0.0.1:8080", WithHTTPTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("NewClient with timeout: %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestNewClient_InvalidHTTPTimeout(t *testing.T) {
	if _, err := NewClient("http://127.0.0.1:8080", WithHTTPTimeout(0)); err == nil {
		t.Fatal("expected error for zero HTTP timeout")
	}
}

var skyflow_client *Client

func TestMain(m *testing.M) {
	var err error
	skyflow_client, err = NewClient(skyflowTestAddress())
	if err != nil {
		panic(err)
	}
	// 运行测试
	code := m.Run()
	os.Exit(code)
}

func skyflowTestAddress() string {
	if address := os.Getenv("SKYFLOW_TEST_SERVER"); address != "" {
		return address
	}
	return "http://127.0.0.1:8080"
}

func TestClient_ConnectionIsNotNil(t *testing.T) {
	if skyflow_client == nil {
		t.Fatal("skyflow_client is nil")
	}
	if skyflow_client.GetClient() == nil {
		t.Fatal("httpClient is nil")
	}
	t.Log("Client connection test passed")
}

func requireSkyflowServer(t *testing.T) {
	t.Helper()
	if os.Getenv("SKYFLOW_TEST_SERVER") == "" {
		t.Skip("set SKYFLOW_TEST_SERVER to run integration tests against a Skyflow server")
	}
}

func TestClient_GetActivityTask(t *testing.T) {
	requireSkyflowServer(t)

	// Test that we can create context and basic request structures
	ctx := context.Background()

	// Test basic request structure creation
	req := &apiV1.GetActivityTaskRequest{
		ActivityUri: "activity:unittest/add",
	}

	// Test GetActivityTask
	resp, err := skyflow_client.GetClient().GetActivityTask(ctx,
		req)

	err2 := errors.FromError(err)
	if err2.Code != 0 {
		t.Fatalf("GetActivityTask failed: %v", err2)
	}
	t.Logf("GetActivityTask response: %v", resp)
}

func TestClient_ListExecutions(t *testing.T) {
	requireSkyflowServer(t)

	ctx := context.Background()
	req := &apiV1.ListExecutionsRequest{
		StatemachineUri: "statemachine:unittest/add",
	}
	resp, err := skyflow_client.GetClient().ListExecutions(ctx, req)
	err2 := errors.FromError(err)
	if err2.Code != 0 {
		t.Fatalf("ListExecutions failed: %v", err2)
	}
	if err != nil {
		t.Fatalf("ListExecutions failed: %v", err)
	}
	t.Logf("ListExecutions response: %v", resp)
}

func TestClient_ListStateMachines(t *testing.T) {
	requireSkyflowServer(t)

	ctx := context.Background()
	req := &apiV1.ListStateMachinesRequest{
		Namespace: "unittest",
	}
	resp, err := skyflow_client.GetClient().ListStateMachines(ctx, req)
	err2 := errors.FromError(err)
	if err2.Code != 0 {
		t.Fatalf("ListStateMachines failed: %v", err2)
	}
	if err != nil {
		t.Fatalf("ListStateMachines failed: %v", err)
	}
	t.Logf("ListStateMachines response: %v", resp)
}

func TestClient_CreateNamespace(t *testing.T) {
	requireSkyflowServer(t)

	ctx := context.Background()
	req := &apiV1.CreateNamespaceRequest{
		Name:        "unittest",
		Description: "unittest description",
	}

	resp, err := skyflow_client.CreateOrUpdateNamespace(ctx, req)
	assert.Equal(t, err, nil)
	assert.Equal(t, resp.Name, "unittest")

}
