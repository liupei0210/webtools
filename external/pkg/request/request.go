package request

import (
	"errors"
	"fmt"
	"github.com/bravpei/webtools/external/pkg/response"
	"github.com/gookit/validate"
	"github.com/kataras/iris/v12"
	log "github.com/sirupsen/logrus"
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
		log.Errorf("Failed to parse parameters: %v", err)
		_ = ctx.JSON(response.Fail(err.Error()))
		return
	}

	// Parameter validation
	if err := validateParams(params); err != nil {
		log.Errorf("Failed to validate parameters: %v", err)
		_ = ctx.JSON(response.Fail(err.Error()))
		return
	}

	// Business logic processing
	data, err := f(params)
	if err != nil {
		log.Errorf("Failed to process business logic: %v", err)
		_ = ctx.JSON(response.Fail(err.Error()))
		return
	}

	_ = ctx.JSON(response.Succeed(data))
}

func parseParams(ctx iris.Context, pType binding, params interface{}) error {
	if pType == NoParam || reflect.TypeOf(params) == nil {
		return nil
	}

	var err error
	switch pType {
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
		return fmt.Errorf("failed to parse %d parameters: %v", pType, err)
	}
	return nil
}

func validateParams(params interface{}) error {
	if v := validate.Struct(params); !v.Validate() {
		return errors.New(v.Errors.One())
	}
	return nil
}
