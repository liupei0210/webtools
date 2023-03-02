package response

type Status int

const (
	Success Status = iota
	Failure
)

type Result struct {
	Status  Status      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func Succeed(data interface{}) Result {
	return Result{
		Status:  Success,
		Message: "success",
		Data:    data,
	}
}
func Fail(message string) Result {
	return Result{
		Status:  Failure,
		Message: message,
	}
}
