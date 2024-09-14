package request

import (
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
func ControllerTemplate[Params interface{}](ctx iris.Context, paramType paramType, f func(p Params) error) {
	var params Params
	if paramType != NoParam && reflect.TypeOf(params) != nil {
		if paramType == BodyParam {
			if err := ctx.ReadBody(&params); err != nil {
				log.Errorf("failed to read JSON: %v", err)
				_ = ctx.JSON(response.Fail(err.Error()))
				return
			}
		} else if paramType == PathParam {
			if err := ctx.ReadParams(&params); err != nil {
				log.Errorf("failed to read JSON: %v", err)
				_ = ctx.JSON(response.Fail(err.Error()))
				return
			}
		} else if paramType == QueryParam {
			if err := ctx.ReadQuery(&params); err != nil {
				log.Errorf("failed to read JSON: %v", err)
				_ = ctx.JSON(response.Fail(err.Error()))
				return
			}
		} else if paramType == FormParam {
			if err := ctx.ReadForm(&params); err != nil {
				log.Errorf("failed to read JSON: %v", err)
				_ = ctx.JSON(response.Fail(err.Error()))
				return
			}
		} else if paramType == HeaderParam {
			if err := ctx.ReadHeaders(&params); err != nil {
				log.Errorf("failed to read JSON: %v", err)
				_ = ctx.JSON(response.Fail(err.Error()))
				return
			}
		}
		v := validate.Struct(params)
		if !v.Validate() {
			log.Error(v.Errors)
			_ = ctx.JSON(response.Fail(v.Errors.One()))
			return
		}
	}
	if err := f(params); err != nil {
		log.Errorf("failed to handle request: %v", err)
		_ = ctx.JSON(response.Fail(err.Error()))
	}
}
