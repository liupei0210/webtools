package request

import (
	"github.com/bravpei/webtools/response"
	"github.com/gookit/validate"
	"github.com/kataras/iris/v12"
	log "github.com/sirupsen/logrus"
)

// ControllerTemplate is a template function for handling requests with JSON parameters.
func ControllerTemplate[Params interface{}](ctx iris.Context, f func(p *Params) error) {
	var params Params
	if err := ctx.ReadJSON(&params); err != nil {
		log.Errorf("failed to read JSON: %v", err)
		_ = ctx.JSON(response.Fail(err.Error()))
		return
	}
	v := validate.Struct(params)
	if !v.Validate() {
		log.Error(v.Errors)
		_ = ctx.JSON(response.Fail(v.Errors.One()))
		return
	}
	if err := f(&params); err != nil {
		log.Errorf("failed to handle request: %v", err)
		_ = ctx.JSON(response.Fail(err.Error()))
	}
}
