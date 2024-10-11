package fornaxplugin

import (
	"context"
	"testing"

	"code.byted.org/flow/eino/schema"
	"code.byted.org/lang/gg/gptr"
	"code.byted.org/overpass/flow_devops_plugin/kitex_gen/flow/devops/plugin/domain/definition"
	"code.byted.org/overpass/flow_devops_plugin/kitex_gen/flow/devops/plugin/domain/tool"
	"github.com/stretchr/testify/assert"
)

func TestConvertToolInfo(t *testing.T) {

	t.Run("success_convert", func(t *testing.T) {
		src := &tool.Tool{
			Name: "update_user_info",
			Desc: "full update user info",
			RequestDefinition: &definition.Definition{
				Type:        "object",
				Description: gptr.Of("ignored desc"),
				Properties: map[string]*definition.Definition{
					"email": {
						Type:        "string",
						Description: gptr.Of("user's email"),
					},
					"job": {
						Type:        "object",
						Description: gptr.Of("user's job"),
						Properties: map[string]*definition.Definition{
							"company": {
								Type:        "string",
								Description: gptr.Of("the company of user's job"),
							},
							"employee_no": {
								Type:        "integer",
								Description: gptr.Of("the number of employee"),
							},
						},
					},
					"incomes": {
						Type:        "array",
						Description: gptr.Of("user's incomes info"),
						Items: &definition.Definition{
							Type:        "object",
							Description: gptr.Of("ignored desc"),
							Properties: map[string]*definition.Definition{
								"source": {
									Type:        "string",
									Description: gptr.Of("the source of income"),
								},
								"amount": {
									Type:        "number",
									Description: gptr.Of("the amount of income"),
								},
							},
						},
					},
				},
				Required: []string{"email", "job", "incomes"},
			},
		}

		expect := &schema.ToolInfo{
			Name: "update_user_info",
			Desc: "full update user info",
			ParamsOneOf: schema.NewParamsOneOfByParams(
				map[string]*schema.ParameterInfo{
					"email": {
						Type:     "string",
						Desc:     "user's email",
						Required: true,
					},
					"job": {
						Type:     "object",
						Desc:     "user's job",
						Required: true,
						SubParams: map[string]*schema.ParameterInfo{
							"company": {
								Type:     "string",
								Desc:     "the company of user's job",
								Required: false,
							},
							"employee_no": {
								Type:     "integer",
								Desc:     "the number of employee",
								Required: false,
							},
						},
					},
					"incomes": {
						Type:     "array",
						Desc:     "user's incomes info",
						Required: true,
						ElemInfo: &schema.ParameterInfo{
							Type:     "object",
							Desc:     "ignored desc",
							Required: false,
							SubParams: map[string]*schema.ParameterInfo{
								"source": {
									Type:     "string",
									Desc:     "the source of income",
									Required: false,
								},
								"amount": {
									Type:     "number",
									Desc:     "the amount of income",
									Required: false,
								},
							},
						},
					},
				}),
		}

		dst, err := convertToolInfo(context.Background(), src)
		assert.NoError(t, err)
		assert.Equal(t, expect, dst)

	})
}
