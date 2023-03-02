package request

import (
	"github.com/gookit/validate"
	"github.com/kataras/iris/v12"
	"github.com/liupei0210/webtools/response"
	log "github.com/sirupsen/logrus"
)

func ControllerTemplate[Params interface{}](ctx iris.Context, f func(p *Params) error) {
	var params Params
	err := ctx.ReadJSON(&params)
	if err != nil {
		log.Error(err.Error())
		_, _ = ctx.JSON(response.Fail(err.Error()))
		return
	}
	validator := validate.New(&params)
	if !validator.Validate() {
		log.Error(validator.Errors)
		_, _ = ctx.JSON(response.Fail(validator.Errors.One()))
		return
	}
	err = f(&params)
	if err != nil {
		log.Error(err.Error())
		_, _ = ctx.JSON(response.Fail(err.Error()))
	}
}
