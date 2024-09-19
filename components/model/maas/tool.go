package maas

type tool struct {
	Function *functionDefinition `json:"function,omitempty"`
}

type functionDefinition struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Parameters  any      `json:"parameters"`
	Examples    []string `json:"examples"`
}
