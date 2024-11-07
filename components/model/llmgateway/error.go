package llmgateway

import "fmt"

type Error struct {
	Message string
	Code    string
}

func (e *Error) Error() string {
	return fmt.Sprintf("GatewayErr: (code: %s, message: %v)", e.Code, e.Message)
}
