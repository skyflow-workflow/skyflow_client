package skyflowclient

import (
	"context"
	"os"
	"testing"

	"github.com/go-kratos/kratos/v2/errors"
	apiV1 "github.com/skyflow-workflow/skyflow_backend/api/v1"
)

var skyflow_client *Client

func TestMain(m *testing.M) {
	var err error
	skyflow_client, err = NewClient("http://127.0.0.1:8080")
	if err != nil {
		panic(err)
	}
	// 运行测试
	code := m.Run()
	os.Exit(code)
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
func TestClient_GetActivityTask(t *testing.T) {
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
	if err != nil {
		t.Fatalf("GetActivityTask failed: %v", err)
	}
	t.Logf("GetActivityTask response: %v", resp)
}

func TestClient_ListExecutions(t *testing.T) {
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
