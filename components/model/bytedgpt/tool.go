package bytedgpt

import (
	"github.com/getkin/kin-openapi/openapi3"
)

type tool struct {
	Function *functionDefinition `json:"function,omitempty"`
}

type functionDefinition struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Parameters  *openapi3.Schema `json:"parameters"`
}
