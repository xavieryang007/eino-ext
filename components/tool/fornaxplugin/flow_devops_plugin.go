package fornaxplugin

import (
	"context"
	"fmt"

	"code.byted.org/flow/eino/schema"
	"code.byted.org/kite/kitex/client/callopt"
	"code.byted.org/kite/kitutil"
	"code.byted.org/lang/gg/gptr"
	"code.byted.org/overpass/flow_devops_plugin/kitex_gen/flow/devops/plugin/domain/definition"
	"code.byted.org/overpass/flow_devops_plugin/kitex_gen/flow/devops/plugin/domain/tool"
	"code.byted.org/overpass/flow_devops_plugin/kitex_gen/flow/devops/plugin/executor"
	"code.byted.org/overpass/flow_devops_plugin/kitex_gen/flow/devops/plugin/fornaxpluginservice"
)

var defaultENV string
var defaultCallOpts []callopt.Option

func InitENV(e string) {
	defaultENV = e
}

func InitCallOpts(opts ...callopt.Option) {
	defaultCallOpts = opts
}

const pluginPSM = "flow.devops.plugin"

var pluginClient fornaxpluginservice.Client

func init() {
	cli, err := fornaxpluginservice.NewClient(pluginPSM)
	if err != nil {
		panic(fmt.Sprintf("init plugin client error: %v", err.Error()))
	}
	pluginClient = cli
}

func getToolDescription(ctx context.Context, toolIDs []int64) (map[int64]*tool.Tool, error) {

	if len(toolIDs) == 0 {
		return map[int64]*tool.Tool{}, nil
	}

	if _, ok := kitutil.GetCtxEnv(ctx); !ok && defaultENV != "" {
		ctx = kitutil.NewCtxWithEnv(ctx, defaultENV)
	}

	req := executor.NewMGetToolDescriptorRequest()
	req.ToolIds = toolIDs

	resp, err := pluginClient.MGetToolDescriptor(ctx, req, defaultCallOpts...)
	if err != nil {
		return nil, fmt.Errorf("MGetToolDescriptor failed: %w", err)
	}

	return resp.GetDescriptors(), nil
}

func executeTool(ctx context.Context, toolID int64, arguments string, opts ...callopt.Option) (string, *definition.Definition, error) {
	req := executor.NewExecuteRequest()
	req.ToolId = gptr.Of(toolID)
	req.Argument = gptr.Of(arguments)

	if _, ok := kitutil.GetCtxEnv(ctx); !ok && defaultENV != "" {
		ctx = kitutil.NewCtxWithEnv(ctx, defaultENV)
	}

	newOpts := make([]callopt.Option, 0, len(defaultCallOpts)+len(opts))
	newOpts = append(newOpts, defaultCallOpts...)
	newOpts = append(newOpts, opts...)
	resp, err := pluginClient.Execute(ctx, req, newOpts...)
	if err != nil {
		return "", nil, err
	}

	return resp.GetData().GetPayload(), resp.GetDescriptor().GetResponseDefinition(), nil
}

func convertToolInfo(_ context.Context, tl *tool.Tool) (*schema.ToolInfo, error) {
	reqDef := tl.GetRequestDefinition()
	if reqDef.GetType() != definition.DataTypeObject {
		return nil, fmt.Errorf("unspported data type: %v", reqDef.GetType())
	}

	topParamInfo, err := convertDefinition(reqDef, false)
	if err != nil {
		return nil, err
	}

	return &schema.ToolInfo{
		Name:        tl.GetName(),
		Desc:        tl.GetDesc(),
		ParamsOneOf: schema.NewParamsOneOfByParams(topParamInfo.SubParams),
	}, nil
}

var typeMapping = map[string]schema.DataType{
	definition.DataTypeNull:    schema.Null,
	definition.DataTypeNumber:  schema.Number,
	definition.DataTypeInteger: schema.Integer,
	definition.DataTypeString:  schema.String,
	definition.DataTypeBoolean: schema.Boolean,
	definition.DataTypeArray:   schema.Array,
	definition.DataTypeObject:  schema.Object,
}

func convertDefinition(def *definition.Definition, required bool) (*schema.ParameterInfo, error) {
	if def == nil {
		return nil, fmt.Errorf("func definition is nil")
	}

	switch def.GetType() {
	case definition.DataTypeNull, definition.DataTypeBoolean, definition.DataTypeInteger,
		definition.DataTypeNumber, definition.DataTypeString:
		return &schema.ParameterInfo{
			Type:     typeMapping[def.GetType()],
			Desc:     def.GetDescription(),
			Required: required,
			Enum:     def.GetEnum(),
		}, nil
	case definition.DataTypeArray:
		elemInfo, err := convertDefinition(def.GetItems(), false)
		if err != nil {
			return nil, err
		}
		return &schema.ParameterInfo{
			Type:     typeMapping[def.GetType()],
			ElemInfo: elemInfo,
			Desc:     def.GetDescription(),
			Required: required,
			Enum:     def.GetEnum(),
		}, nil
	case definition.DataTypeObject:
		requireFields := make(map[string]bool, len(def.GetRequired()))
		params := make(map[string]*schema.ParameterInfo, len(def.GetProperties()))
		for _, fieldName := range def.GetRequired() {
			requireFields[fieldName] = true
		}

		for fieldName, subDef := range def.GetProperties() {
			pi, err := convertDefinition(subDef, requireFields[fieldName])
			if err != nil {
				return nil, err
			}
			params[fieldName] = pi
		}

		return &schema.ParameterInfo{
			Type:      typeMapping[def.GetType()],
			Desc:      def.GetDescription(),
			SubParams: params,
			Required:  required,
			Enum:      def.GetEnum(),
		}, nil

	}

	return nil, fmt.Errorf("unspported data type: %v", def.GetType())
}
