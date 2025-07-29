package request

import (
	"errors"
	"github.com/gookit/validate"
	"github.com/kataras/iris/v12"
	"github.com/liupei0210/webtools/external/pkg/response"
	"github.com/liupei0210/webtools/external/pkg/utils"
)

type binding int8

const (
	NoParam binding = iota
	BodyParam
	PathParam
	QueryParam
	FormParam
	HeaderParam
)

// ControllerTemplate is a template function for handling requests with JSON parameters.
func ControllerTemplate[Params interface{}](ctx iris.Context, binding binding, f func(p Params) (interface{}, error)) {
	var params Params

	// Parameter parsing
	if err := ctx.ReadBody(&params); err != nil {
		utils.GetLogger().Errorf("Failed to parse parameters: %v", err)
		_ = ctx.JSON(response.Fail(err.Error()))
		return
	}

	// Parameter validation
	if err := validateParams(params); err != nil {
		utils.GetLogger().Errorf("Failed to validate parameters: %v", err)
		_ = ctx.JSON(response.Fail(err.Error()))
		return
	}

	// Business logic processing
	data, err := f(params)
	if err != nil {
		utils.GetLogger().Errorf("Failed to process business logic: %v", err)
		_ = ctx.JSON(response.Fail(err.Error()))
		return
	}

	_ = ctx.JSON(response.Succeed(data))
}

func validateParams(params interface{}) error {
	if v := validate.Struct(params); !v.Validate() {
		return errors.New(v.Errors.One())
	}
	return nil
}
