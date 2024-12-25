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

type paramType int8

const (
	NoParam paramType = iota
	BodyParam
	PathParam
	QueryParam
	FormParam
	HeaderParam
)

// ControllerTemplate is a template function for handling requests with JSON parameters.
func ControllerTemplate[Params interface{}](ctx iris.Context, paramType paramType, f func(p Params) (interface{}, error)) {
	var params Params

	// 参数解析
	if err := parseParams(ctx, paramType, &params); err != nil {
		log.Errorf("参数解析失败: %v", err)
		_ = ctx.JSON(response.Fail(err.Error()))
		return
	}

	// 参数验证
	if err := validateParams(params); err != nil {
		log.Errorf("参数验证失败: %v", err)
		_ = ctx.JSON(response.Fail(err.Error()))
		return
	}

	// 业务处理
	data, err := f(params)
	if err != nil {
		log.Errorf("业务处理失败: %v", err)
		_ = ctx.JSON(response.Fail(err.Error()))
		return
	}

	_ = ctx.JSON(response.Succeed(data))
}

func parseParams(ctx iris.Context, pType paramType, params interface{}) error {
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
	}

	if err != nil {
		return fmt.Errorf("解析%s参数失败: %v", pType, err)
	}
	return nil
}

func validateParams(params interface{}) error {
	if v := validate.Struct(params); !v.Validate() {
		return errors.New(v.Errors.One())
	}
	return nil
}
