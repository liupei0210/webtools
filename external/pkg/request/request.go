package request

import (
	"errors"
	"fmt"
	"github.com/bravpei/webtools/external/pkg/response"
	"github.com/bravpei/webtools/external/pkg/utils"
	"github.com/gookit/validate"
	"github.com/kataras/iris/v12"
	"reflect"
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
	if err := parseParams(ctx, binding, &params); err != nil {
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

func parseParams(ctx iris.Context, binding binding, params interface{}) error {
	if binding == NoParam || reflect.TypeOf(params) == nil {
		return nil
	}

	var err error
	switch binding {
	case BodyParam:
		err = ctx.ReadBody(params)
	case PathParam:
		err = ctx.ReadParams(params)
	case QueryParam:
		err = ctx.ReadQuery(params)
	case FormParam:
		err = ctx.ReadForm(params)
	case HeaderParam:
		err = ctx.ReadHeaders(params)
	default:
		err = errors.New("unhandled default case")
	}

	if err != nil {
		return fmt.Errorf("failed to parse %d parameters: %v", binding, err)
	}
	return nil
}

func validateParams(params interface{}) error {
	if v := validate.Struct(params); !v.Validate() {
		return errors.New(v.Errors.One())
	}
	return nil
}
