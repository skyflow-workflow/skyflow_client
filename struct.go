package skyflowclient

// ResponseMessage 返回结构体
type ResponseMessage struct {
	ErrorCode    string      `json:"ErrorCode"`
	ErrorMessage string      `json:"ErrorMessage"`
	Success      bool        `json:"Success"`
	Data         interface{} `json:"Data"`
}

type StartExecutionRequest struct {
	StatemachineUri string `json:"state_machine_uri"`
	Input           any    `json:"input"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	ExecutionName   string `json:"execution_name"`
	Definition      string `json:"definition"`
}
