package response

type Status int

const (
	success Status = iota
	failure
	validationError
	serverError
)

type Result struct {
	Status  Status      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	TraceId string      `json:"trace_id,omitempty"`
}

var statusMessages = map[Status]string{
	success:         "操作成功",
	failure:         "操作失败",
	validationError: "参数验证失败",
	serverError:     "服务器内部错误",
}

func Succeed(data interface{}) Result {
	return Result{
		Status:  success,
		Message: statusMessages[success],
		Data:    data,
	}
}

func Fail(message string) Result {
	return Result{
		Status:  failure,
		Message: message,
	}
}

func ValidateError(message string) Result {
	return Result{
		Status:  validationError,
		Message: message,
	}
}

func ServerError(err error) Result {
	return Result{
		Status:  serverError,
		Message: statusMessages[serverError],
		Data:    err.Error(),
	}
}
