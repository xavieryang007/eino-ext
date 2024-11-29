package utils

import (
	"encoding/json"
	"testing"

	"code.byted.org/flow/eino/schema"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
)

func TestToGWTool(t *testing.T) {
	einoToolInfo := &schema.ToolInfo{
		Name: "System_UserSelect",
		Desc: "Show select UI and let user select one from multiple entity candidates.",
		ParamsOneOf: schema.NewParamsOneOfByOpenAPIV3(&openapi3.Schema{
			Type: openapi3.TypeObject,
			Properties: openapi3.Schemas{
				"candidate": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: openapi3.TypeArray,
						Items: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type:        openapi3.TypeString,
								Description: "entity candidates list, every entity is represented by its ID value, not id value.",

								Properties: make(openapi3.Schemas),
								Extensions: make(map[string]interface{}),
							},
						},
						Properties: make(openapi3.Schemas),
						Extensions: make(map[string]interface{}),
					},
				},
				"type": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:        openapi3.TypeString,
						Description: "the type name of these entity candidates, must choose one from following enums [contact,callHistory,photo,album,note,message].",
						Properties:  make(openapi3.Schemas),
						Extensions:  make(map[string]interface{}),
					},
				},
			},
			Extensions: make(map[string]interface{}),
		}),
	}

	einoSchema, err := einoToolInfo.ToOpenAPIV3()
	assert.NoError(t, err)

	t.Run("openapi3_schema_to_gw_schema", func(t *testing.T) {
		gwToolInfo, err := toGWTool(einoToolInfo)
		assert.NoError(t, err)

		// data, _ := json.MarshalIndent(gwToolInfo, "", "  ")
		//
		// t.Logf("json schema: \n%s", string(data))

		gwToolInfoData, err := json.Marshal(gwToolInfo.Function_.Parameters)
		assert.NoError(t, err)
		gwSchemaReversed := &openapi3.Schema{}
		err = json.Unmarshal(gwToolInfoData, gwSchemaReversed)
		assert.NoError(t, err)

		assert.Equal(t, einoSchema, gwSchemaReversed)
	})

	t.Run("parameter_info_to_gw_schema", func(t *testing.T) {

		einoToolInfoBasedOnParameterInfo := &schema.ToolInfo{
			Name: "System_UserSelect",
			Desc: "Show select UI and let user select one from multiple entity candidates.",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"candidate": {
					Type: schema.Array,
					ElemInfo: &schema.ParameterInfo{
						Type: schema.String,
						Desc: "entity candidates list, every entity is represented by its ID value, not id value.",
					},
				},
				"type": {
					Type: schema.String,
					Desc: "the type name of these entity candidates, must choose one from following enums [contact,callHistory,photo,album,note,message].",
				},
			},
			),
		}

		gwToolInfo, err := toGWTool(einoToolInfoBasedOnParameterInfo)
		assert.NoError(t, err)

		// data, _ := json.MarshalIndent(gwToolInfo, "", "  ")
		// t.Logf("json schema: \n%s", string(data))

		gwToolInfoData, err := json.Marshal(gwToolInfo.Function_.Parameters)
		assert.NoError(t, err)

		gwSchemaReversed := &openapi3.Schema{}
		err = json.Unmarshal(gwToolInfoData, gwSchemaReversed)
		assert.NoError(t, err)

		assert.Equal(t, einoSchema, gwSchemaReversed)
	})
}
