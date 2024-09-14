package request

import "testing"

type Abc struct {
}

func TestControllerTemplate(t *testing.T) {
	ControllerTemplate[any](nil, NoParam, func(p any) error {
		return nil
	})
}
