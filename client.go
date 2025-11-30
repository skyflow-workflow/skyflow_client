package skyflowclient

import (
	"github.com/go-kratos/kratos/v2/internal/httputil"
	apiV1 "github.com/skyflow-workflow/skyflow_backend/api/v1"
)

type Client struct {
	client apiV1.SkyflowV1ServiceHTTPClient
}

func NewClient(address string) (*Client, error) {

	client := apiV1.NewSkyflowV1ServiceHTTPClient(
		httputil.WithEndpoint(address),
	)
	return &Client{
		client: client,
	}, nil
}

func (client *Client) Close() error {
	return nil
}

func (client *Client) GetClient() apiV1.SkyflowV1ServiceHTTPClient {
	return client.client
}

func (client *Client) SendTaskFailure(token string) {
	httputil.SetToken(client.client, token)
}

func (client *Client) SendTaskSuccess(token string) {
	client.GetClient().SendTaskSuccess(token, map[string]interface{}{})
}
